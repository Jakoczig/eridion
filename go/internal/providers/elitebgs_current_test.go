package providers

import (
	"net/url"
	"strings"
	"testing"
)

// Task 17: on the Elite BGS v5 routes (elite-kode/elitebgs, server/routes/
// elite_bgs_api/v5/{systems,factions,stations}.js at d458e6fc) `count` is
// "Number of history records" and switches the reply to history mode. A
// current lookup must therefore carry only the name and the page.
func TestAuditEliteBGSCurrentLookupExcludesHistory(t *testing.T) {
	for _, kind := range []string{"systems", "factions", "stations"} {
		raw, err := communityEndpoint("elitebgs", kind, "Fixture Alpha", 2)
		if err != nil {
			t.Fatal(kind, err)
		}
		u, err := url.Parse(raw)
		if err != nil || !allowedEndpoint(u) {
			t.Fatal("documented current lookup denied", raw)
		}
		q := u.Query()
		if q.Get("name") != "Fixture Alpha" || q.Get("page") != "2" {
			t.Fatal("current lookup lost name/page", raw)
		}
		for _, history := range []string{"count", "timeMin", "timeMax"} {
			if q.Has(history) {
				t.Fatalf("%s lookup selects historical records via %s: %s", kind, history, raw)
			}
		}
		for _, page := range []int{0, 11} {
			if _, err := communityEndpoint("elitebgs", kind, "Fixture Alpha", page); err == nil {
				t.Fatal("page outside 1..10 accepted", kind, page)
			}
		}
	}
}

func TestAuditEliteBGSCurrentResponses(t *testing.T) {
	facts := func(s *Snapshot) map[string]string {
		m := map[string]string{}
		for _, f := range s.Facts {
			m[f.Name] = f.Value
		}
		return m
	}
	// Current: one exact entity with its current state.
	s, err := normalizeCommunity("elitebgs", "systems", "Fixture Alpha", []byte(`{"docs":[{"name":"Fixture Alpha","system_address":"9007199254740993","government":"$government_democracy;","controlling_minor_faction_cased":"Fixture Union","updated_at":"2026-11-20T00:00:00Z"}],"total":1,"page":1,"pages":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if f := facts(s); f["1 / system_address"] != "9007199254740993" || f["1 / controlling_minor_faction_cased"] != "Fixture Union" || f["pages"] != "1" {
		t.Fatal("current state not normalized", f)
	}
	// Empty: an unknown name is a successful, empty page, never an invented entity.
	s, err = normalizeCommunity("elitebgs", "factions", "Fixture Nobody", []byte(`{"docs":[],"total":0,"page":1,"pages":1}`))
	if err != nil {
		t.Fatal(err)
	}
	for name := range facts(s) {
		if strings.HasPrefix(name, "1 / ") {
			t.Fatal("empty page produced entity facts", name)
		}
	}
	if facts(s)["total"] != "0" {
		t.Fatal("empty page total missing", s.Facts)
	}
	// Errors: error envelopes, non-JSON and oversized pages are refused.
	for _, raw := range []string{`{"error":"Internal Server Error"}`, `<html>502</html>`, `{"docs":"x"}`, `{"docs":[` + strings.Repeat(`{"name":"Fixture Alpha"},`, 32) + `{"name":"Fixture Alpha"}]}`} {
		if _, err := normalizeCommunity("elitebgs", "stations", "Fixture Alpha", []byte(raw)); err == nil {
			t.Fatal("error reply accepted", raw[:min(len(raw), 40)])
		}
	}
	// Pagination: the page position is reported so the caller can request the next page.
	s, err = normalizeCommunity("elitebgs", "stations", "Fixture Port", []byte(`{"docs":[{"name":"Fixture Port","market_id":"3228000000"}],"total":12,"page":2,"pages":3}`))
	if err != nil {
		t.Fatal(err)
	}
	if f := facts(s); f["page"] != "2" || f["pages"] != "3" || f["total"] != "12" || f["1 / market_id"] != "3228000000" {
		t.Fatal("pagination facts", f)
	}
}
