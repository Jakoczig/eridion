package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Jakoczig/eridion/go/internal/privatefs"
)

func accountKey(fid string) string {
	h := sha256.Sum256([]byte(fid))
	return hex.EncodeToString(h[:])
}

func enabledConsent(categories ...string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"read": true, "upload": true, "paused": false, "categories": categories, "futureField": "kept"})
	return raw
}

// Restore must require renewed consent for every account namespace in the
// provider ledger, whichever account was active or pinned when backed up.
func TestAuditRestoreAllAccounts(t *testing.T) {
	a, b, pinned := accountKey("SYNTHETIC-A"), accountKey("SYNTHETIC-B"), accountKey("SYNTHETIC-C")
	for _, active := range []string{a, b, pinned} {
		t.Run(active[:8], func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.writer.Exec("INSERT INTO commander VALUES('SYNTHETIC-A','A',?),('SYNTHETIC-B','B',?)", utc(), utc()); err != nil {
				t.Fatal(err)
			}
			for _, fid := range []string{"SYNTHETIC-A", "SYNTHETIC-B"} {
				if err = s.SetPublicationPolicy(fid, "After sale"); err != nil {
					t.Fatal(err)
				}
				if err = s.SetLiveException(fid, "eddn", "travel", true, true); err != nil {
					t.Fatal(err)
				}
			}
			disk := ProviderDisk{
				ActiveAccount: active,
				Providers:     map[string]json.RawMessage{"edsm": enabledConsent("travel")},
				Accounts: map[string]map[string]json.RawMessage{
					a: {"edsm": enabledConsent("travel"), "inara": enabledConsent("ranks")},
					b: {"edsm": enabledConsent("exploration")},
				},
				Ledger: map[string]int64{"edsm:dedupe": 7},
			}
			raw, _ := json.Marshal(disk)
			if err = s.SaveProviderConsent(raw); err != nil {
				t.Fatal(err)
			}
			backupDir := t.TempDir()
			privatefs.Prepare(backupDir)
			backup := filepath.Join(backupDir, "fixture.sqlite")
			if _, err = s.Backup(context.Background(), backup); err != nil {
				t.Fatal(err)
			}
			if err = s.StageRestore(context.Background(), backup); err != nil {
				t.Fatal(err)
			}
			s.Close()
			for restart := 0; restart < 2; restart++ {
				s, err = Open(dir)
				if err != nil {
					t.Fatal(err)
				}
				loaded, err := s.LoadProviderConsent(nil)
				if err != nil {
					t.Fatal(err)
				}
				var got ProviderDisk
				if err = json.Unmarshal(loaded, &got); err != nil {
					t.Fatal(err)
				}
				check := func(scope string, cfg json.RawMessage) {
					var c map[string]any
					if err := json.Unmarshal(cfg, &c); err != nil {
						t.Fatal(err)
					}
					if c["upload"] != false || c["paused"] != true {
						t.Fatalf("%s consent revived after restore: %s", scope, cfg)
					}
					if c["futureField"] != "kept" || c["categories"] == nil {
						t.Fatalf("%s lost unrelated configuration: %s", scope, cfg)
					}
				}
				for id, cfg := range got.Providers {
					check("active/"+id, cfg)
				}
				if len(got.Accounts) != 2 {
					t.Fatalf("accounts changed: %v", got.Accounts)
				}
				for account, providers := range got.Accounts {
					for id, cfg := range providers {
						check(account[:8]+"/"+id, cfg)
					}
				}
				if got.ActiveAccount != active || got.Ledger["edsm:dedupe"] != 7 {
					t.Fatalf("ledger identity/dedupe not retained: %+v", got)
				}
				for _, fid := range []string{"SYNTHETIC-A", "SYNTHETIC-B"} {
					policy, err := s.PublicationPolicy(fid)
					if err != nil || policy.Mode != "Never" || len(policy.Exceptions) != 0 {
						t.Fatalf("%s policy after restore: %+v %v", fid, policy, err)
					}
				}
				s.Close()
			}
		})
	}
}
