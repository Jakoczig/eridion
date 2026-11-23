package providers

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/history"
	"github.com/Jakoczig/eridion/go/internal/journal"
)

type receiptFixture struct {
	m      *Manager
	db     *history.Store
	now    *time.Time
	events []journal.Event
	calls  *int
	t      *testing.T
}

// edsmReceiptFixture queues two live EDSM travel events for one batch; reply
// answers the n-th request.
func edsmReceiptFixture(t *testing.T, dir, profile string, reply func(n int) (int, string)) *receiptFixture {
	t.Helper()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	db, err := history.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	calls := 0
	opts := Options{RequireHistory: true, Manual: true, Vault: &testVault{}, Now: func() time.Time { return now }, Client: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		status, body := reply(calls)
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	m := New(context.Background(), profile, opts)
	t.Cleanup(func() { m.Close() })
	if err = m.AttachHistory(db, history.ProviderSeed{}); err != nil {
		t.Fatal(err)
	}
	f := &receiptFixture{m: m, db: db, now: &now, calls: &calls, t: t}
	src := journal.Source{ID: "edsm-receipts", Identity: "synthetic"}
	cp, err := db.Resume(src)
	if err != nil {
		t.Fatal(err)
	}
	add := func(raw string) journal.Event {
		e, err := journal.ParseLine([]byte(raw), journal.SourcePosition{FileID: src.ID, Sequence: cp.Sequence + 1, Offset: cp.Offset, Length: 100})
		if err != nil {
			t.Fatal(err)
		}
		cp.Sequence++
		cp.Offset += 100
		if err = db.Commit(context.Background(), cp, []journal.Record{{Event: e, Phase: journal.Live}}, nil); err != nil {
			t.Fatal(err)
		}
		m.Observe(e, journal.Live)
		return e
	}
	add(`{"event":"Fileheader","timestamp":"2026-11-15T11:59:00Z","gameversion":"4.1"}`)
	add(`{"event":"LoadGame","timestamp":"2026-11-15T11:59:01Z","FID":"SYNTHETIC-A","Commander":"Fixture"}`)
	if err = m.SetCredential("edsm", Credential{APIKey: "SYNTHETIC", Commander: "Fixture"}); err != nil {
		t.Fatal(err)
	}
	if err = m.Configure("edsm", Config{Upload: true, Categories: []string{"travel"}}); err != nil {
		t.Fatal(err)
	}
	if err = m.SetPublicationPolicy("SYNTHETIC-A", "Live"); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		f.events = append(f.events, add(`{"event":"FSDJump","timestamp":"2026-11-15T11:59:5`+strconv.Itoa(i)+`Z","StarSystem":"Fixture `+strconv.Itoa(i)+`","SystemAddress":900719925474099`+strconv.Itoa(i)+`,"StarPos":[1,2,3],"JumpDist":12}`))
	}
	if q := len(m.states["edsm"].queue); q != 2 {
		t.Fatalf("fixture queued %d events", q)
	}
	return f
}

// status reads the durable outcome. Candidate re-evaluates and upserts the
// row, so a terminal outcome must survive that re-evaluation.
func (f *receiptFixture) status(i int) string {
	f.t.Helper()
	c, err := f.db.Candidate("SYNTHETIC-A", "edsm", "travel", f.events[i], journal.Live, false, *f.now)
	if err != nil {
		f.t.Fatal(err)
	}
	return c.Status
}

// Task 7 / F08: every submitted EDSM event gets a truthful, durable outcome
// from the per-event receipts; only genuine confirmation counts as sent.
func TestAuditReceiptMatrix(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   [2]string
		sent   uint64
	}{
		{"mixed per-event results", 200, `{"msgnum":100,"msg":"OK","events":[{"msgnum":100,"msg":"OK"},{"msgnum":402,"msg":"Item unknown"}]}`, [2]string{"confirmed", "rejected"}, 1},
		{"outer success, inner fatal", 200, `{"msgnum":100,"msg":"OK","events":[{"msgnum":203,"msg":"x"},{"msgnum":304,"msg":"x"}]}`, [2]string{"rejected", "rejected"}, 0},
		{"short receipt list", 200, `{"msgnum":100,"msg":"OK","events":[{"msgnum":100,"msg":"OK"}]}`, [2]string{"unknown_delivery", "unknown_delivery"}, 0},
		{"extra receipts", 200, `{"msgnum":100,"msg":"OK","events":[{"msgnum":100},{"msgnum":100},{"msgnum":100}]}`, [2]string{"unknown_delivery", "unknown_delivery"}, 0},
		{"no per-event receipts", 200, `{"msgnum":100,"msg":"OK"}`, [2]string{"unknown_delivery", "unknown_delivery"}, 0},
		{"batch refused", 200, `{"msgnum":203,"msg":"Commander name/API Key not found"}`, [2]string{"rejected", "rejected"}, 0},
		{"saved for later", 200, `{"msgnum":500,"msg":"Saved for later"}`, [2]string{"accepted_for_later", "accepted_for_later"}, 0},
		{"unreadable reply", 200, `not json`, [2]string{"unknown_delivery", "unknown_delivery"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, profile := t.TempDir(), t.TempDir()
			f := edsmReceiptFixture(t, dir, profile, func(int) (int, string) { return tc.status, tc.body })
			f.m.dispatch("edsm")
			if *f.calls != 1 {
				t.Fatalf("%d requests", *f.calls)
			}
			for i := range f.events {
				if got := f.status(i); got != tc.want[i] {
					t.Fatalf("event %d outcome %q, want %q", i, got, tc.want[i])
				}
			}
			s := f.m.states["edsm"]
			if s.sent != tc.sent || s.counts["travel"] != tc.sent || len(s.queue) != 0 {
				t.Fatalf("sent=%d travel=%d queue=%d", s.sent, s.counts["travel"], len(s.queue))
			}
			// No automatic resend of any of these outcomes.
			*f.now = f.now.Add(time.Hour)
			f.m.dispatch("edsm")
			if *f.calls != 1 {
				t.Fatal("outcome resent automatically")
			}
		})
	}
}

// Task 7 / F08: repeated 429/503 refusals end in a terminal failed outcome with
// an empty queue and no fifth request; outcomes survive a restart and are not
// re-queued when the same events are evaluated again.
func TestAuditRefusalExhaustion(t *testing.T) {
	for _, code := range []int{429, 503} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			dir, profile := t.TempDir(), t.TempDir()
			f := edsmReceiptFixture(t, dir, profile, func(int) (int, string) { return code, `` })
			for i := 0; i < 6; i++ {
				f.m.dispatch("edsm")
				*f.now = f.now.Add(time.Hour)
			}
			if *f.calls != 4 {
				t.Fatalf("%d requests, want 4", *f.calls)
			}
			if q := len(f.m.states["edsm"].queue); q != 0 {
				t.Fatalf("queue residue %d", q)
			}
			for i := range f.events {
				if got := f.status(i); got != "failed" {
					t.Fatalf("event %d outcome %q", i, got)
				}
			}
			// Restart: the durable outcome is kept and nothing is re-queued.
			f.m.Close()
			f.db.Close()
			db, err := history.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			for i := range f.events {
				c, err := db.Candidate("SYNTHETIC-A", "edsm", "travel", f.events[i], journal.Live, false, *f.now)
				if err != nil || c.Status != "failed" {
					t.Fatalf("after restart event %d = %q %v", i, c.Status, err)
				}
			}
		})
	}
}
