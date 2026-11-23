package history

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Source authority (confirmed owner decision D0):
//
//  1. fresh, validated Frontier evidence;
//  2. valid journal / attributable Status evidence;
//  3. Inara and every other third-party provider, at one equal tier.
//
// Authority is decided per field (and per nested group by the callers that
// resolve groups, such as ResolveRanks). Chronology inside one source is
// decided earlier by currentFactEvidence; this resolver never reintroduces a
// cross-source preference based on observation times. Evidence that is not
// eligible (expired, unattributable or with an impossible observation time)
// stays available as a labelled fallback below every eligible candidate.
const (
	tierFrontier   = 0
	tierJournal    = 1
	tierThirdParty = 2
)

func sourceTier(source string) int {
	switch source {
	case "frontier":
		return tierFrontier
	case "journal":
		return tierJournal
	default:
		return tierThirdParty
	}
}

func authorityLabel(tier int) string {
	switch tier {
	case tierFrontier:
		return "frontier"
	case tierJournal:
		return "journal"
	default:
		return "third-party"
	}
}

// eligibleEvidence reports whether v may take part in its tier. Frontier CAPI
// usually has no observation time; freshness comes from the real fetch/expiry
// (already folded into Stale). A reported observation time must still be
// possible: not in the future and not after the fetch that returned it.
func eligibleEvidence(v FactEvidence, now time.Time) bool {
	switch sourceTier(v.Source) {
	case tierJournal:
		_, ok := validObservedAt(v.Observed, now)
		return ok
	default:
		if v.Stale {
			return false
		}
		if v.Observed == "" {
			return true
		}
		observed, ok := validObservedAt(v.Observed, now)
		if !ok {
			return false
		}
		if fetched, ok := parseObservation(v.Fetched); ok && observed.After(fetched.Add(futureTolerance)) {
			return false
		}
		return true
	}
}

func referenceNumber(ref string) int64 {
	n, _ := strconv.ParseInt(ref, 10, 64)
	return n
}

// ResolveEvidence orders candidates for one field by authority and annotates
// each with its tier and eligibility. The first element is the effective
// value; the rest are retained alternatives. Within a tier the newest
// plausible observation wins, then the latest fetch, then the later stored
// reference. Source names never decide inside the equal third-party tier;
// only fully identical candidates fall back to a name comparison so that the
// order stays deterministic.
func ResolveEvidence(candidates []FactEvidence, now time.Time) []FactEvidence {
	cutoff := observationCutoff(now)
	for i := range candidates {
		tier := sourceTier(candidates[i].Source)
		candidates[i].Authority = authorityLabel(tier)
		candidates[i].Eligible = eligibleEvidence(candidates[i], now)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.Eligible != b.Eligible {
			return a.Eligible
		}
		if ta, tb := sourceTier(a.Source), sourceTier(b.Source); ta != tb {
			return ta < tb
		}
		ka, kb := observationKey(a.Observed), observationKey(b.Observed)
		pa, pb := ka != "" && ka <= cutoff, kb != "" && kb <= cutoff
		if pa != pb {
			return pa
		}
		if ka != kb {
			return ka > kb
		}
		fa, fb := observationKey(a.Fetched), observationKey(b.Fetched)
		if fa != fb {
			return fa > fb
		}
		if ra, rb := referenceNumber(a.Reference), referenceNumber(b.Reference); ra != rb {
			return ra > rb
		}
		return a.Source < b.Source
	})
	return candidates
}

// ResolvedRank is one pilot-rank category resolved under the D0 order. A
// progress percentage is only reported when it belongs to the same level as
// the effective rank; otherwise it stays unknown.
type ResolvedRank struct {
	Category       string  `json:"category"`
	Rank           string  `json:"rank"`
	Source         string  `json:"source"`
	Authority      string  `json:"authority"`
	ObservedAt     string  `json:"observedAt"`
	FetchedAt      string  `json:"fetchedAt"`
	Stale          bool    `json:"stale"`
	Progress       *string `json:"progress"`
	ProgressSource string  `json:"progressSource,omitempty"`
	// Alternatives keeps lower-authority levels that were replaced, so
	// provenance stays visible (for example an Inara or expired Frontier
	// level superseded by valid journal evidence).
	Alternatives []RankAlternative `json:"alternatives,omitempty"`
}

// RankAlternative is a superseded rank level with its source labels.
type RankAlternative struct {
	Rank      string `json:"rank"`
	Source    string `json:"source"`
	Authority string `json:"authority"`
	FetchedAt string `json:"fetchedAt,omitempty"`
	Stale     bool   `json:"stale"`
}

var rankCategories = []string{"combat", "trade", "explore", "cqc", "empire", "federation", "soldier", "exobiologist", "power", "builder", "learner"}

func rankCategory(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "exploration", "explorer":
		return "explore"
	case "mercenary":
		return "soldier"
	case "exobiology":
		return "exobiologist"
	}
	for _, c := range rankCategories {
		if c == name {
			return c
		}
	}
	return ""
}

func rankText(v any) string {
	switch x := v.(type) {
	case string:
		if identifier(x) != "" {
			return x
		}
	case float64:
		if x >= 0 && x <= 100 && x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
	case json.Number:
		return rankText(string(x))
	}
	return ""
}

// rankLevels normalizes the journal (Rank event object), Frontier (lowercase
// object) and Inara (name/value list) shapes to category -> level text.
func rankLevels(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil {
		for k, v := range obj {
			if c := rankCategory(k); c != "" {
				if level := rankText(v); level != "" {
					out[c] = level
				}
			}
		}
		return out
	}
	var list []map[string]any
	if json.Unmarshal(raw, &list) == nil {
		for _, item := range list {
			name, _ := item["name"].(string)
			if c := rankCategory(name); c != "" {
				if level := rankText(item["value"]); level != "" {
					out[c] = level
				}
			}
		}
	}
	return out
}

func factCandidates(facts []EffectiveFact, field string) []FactEvidence {
	for _, f := range facts {
		if f.Field == field {
			return append([]FactEvidence{f.FactEvidence}, f.Alternatives...)
		}
	}
	return nil
}

// ResolveRanks resolves every rank category independently: a source missing
// one category does not clear it, and journal progress is attached only when
// the journal level equals the effective level and the progress observation
// is not older than that journal level.
func ResolveRanks(facts []EffectiveFact) []ResolvedRank {
	candidates := factCandidates(facts, "ranks")
	var journalLevels map[string]string
	var journalRankKey string
	for _, c := range candidates {
		if c.Source == "journal" {
			journalLevels, journalRankKey = rankLevels(c.Value), observationKey(c.Observed)
			break
		}
	}
	var progress map[string]string
	var progressKey string
	for _, c := range factCandidates(facts, "rankProgress") {
		if c.Source == "journal" {
			progress, progressKey = rankLevels(c.Value), observationKey(c.Observed)
			break
		}
	}
	out := []ResolvedRank{}
	for _, category := range rankCategories {
		for _, c := range candidates {
			level, ok := rankLevels(c.Value)[category]
			if !ok {
				continue
			}
			r := ResolvedRank{Category: category, Rank: level, Source: c.Source, Authority: c.Authority, ObservedAt: c.Observed, FetchedAt: c.Fetched, Stale: !c.Eligible || c.Stale}
			if p, ok := progress[category]; ok && journalLevels[category] == level && progressKey >= journalRankKey {
				value := p
				r.Progress, r.ProgressSource = &value, "journal"
			}
			out = append(out, r)
			break
		}
	}
	return out
}

// CurrentShipIdentity is the commander's current ship as reported by eligible
// Frontier evidence. Journal reducers remain the fallback owned by callers.
type CurrentShipIdentity struct {
	ID, Name, Model, SystemName, Source, FetchedAt string
}

// ResolveCurrentShip returns the Frontier current ship when the effective
// "ship" fact is fresh, validated Frontier evidence that CAPI itself marked as
// the commander's current ship. Anything else leaves the journal view in place.
func ResolveCurrentShip(facts []EffectiveFact) (CurrentShipIdentity, bool) {
	for _, f := range facts {
		if f.Field != "ship" || f.Source != "frontier" || !f.Eligible {
			continue
		}
		var v struct {
			ID               string `json:"id"`
			Model            string `json:"model"`
			Name             string `json:"name"`
			SystemName       string `json:"systemName"`
			CurrentReference bool   `json:"currentReference"`
		}
		if json.Unmarshal(f.Value, &v) != nil || !v.CurrentReference || identifier(v.ID) == "" {
			return CurrentShipIdentity{}, false
		}
		return CurrentShipIdentity{ID: v.ID, Name: v.Name, Model: v.Model, SystemName: v.SystemName, Source: "frontier", FetchedAt: f.Fetched}, true
	}
	return CurrentShipIdentity{}, false
}

// JournalRank is a rank level/progress pair from the live journal reducer
// (exact decimal text; nil means unknown).
type JournalRank struct {
	Category       string
	Rank, Progress *string
}

// MergeJournalRanks applies the D0 order to journal-tier evidence known only
// to the live reducer. Real journals write Rank/Progress before LoadGame, so
// those levels never reach commander provenance; the reducer has already
// attributed them to the commander and applied within-source chronology. A
// valid journal level therefore replaces a third-party level, an expired or
// otherwise ineligible Frontier level and an older journal level from
// provenance. Only an eligible (fresh, validated) Frontier level keeps
// precedence. Replaced levels stay as labelled alternatives, and journal
// progress attaches only to the same level.
func MergeJournalRanks(resolved []ResolvedRank, journal []JournalRank) []ResolvedRank {
	out := append([]ResolvedRank{}, resolved...)
	for _, j := range journal {
		category := rankCategory(j.Category)
		if category == "" || j.Rank == nil || *j.Rank == "" {
			continue
		}
		live := ResolvedRank{Category: category, Rank: *j.Rank, Source: "journal", Authority: "journal"}
		if j.Progress != nil {
			progress := *j.Progress
			live.Progress, live.ProgressSource = &progress, "journal"
		}
		found := false
		for i := range out {
			if out[i].Category != category {
				continue
			}
			found = true
			current := out[i]
			if current.Source == "frontier" && !current.Stale {
				// Eligible Frontier wins; journal progress only for the same level.
				if current.Progress == nil && live.Progress != nil && current.Rank == live.Rank {
					out[i].Progress, out[i].ProgressSource = live.Progress, "journal"
				}
				continue
			}
			if current.Source == "journal" {
				// Same source: keep provenance labels when the level agrees.
				if current.Rank == live.Rank {
					if live.Progress != nil {
						out[i].Progress, out[i].ProgressSource = live.Progress, "journal"
					}
					out[i].Stale = false
					continue
				}
				live.ObservedAt = ""
			}
			live.Alternatives = append(append([]RankAlternative{}, current.Alternatives...), RankAlternative{Rank: current.Rank, Source: current.Source, Authority: current.Authority, FetchedAt: current.FetchedAt, Stale: current.Stale})
			out[i] = live
		}
		if !found {
			out = append(out, live)
		}
	}
	index := map[string]int{}
	for i, c := range rankCategories {
		index[c] = i
	}
	sort.SliceStable(out, func(a, b int) bool { return index[out[a].Category] < index[out[b].Category] })
	return out
}
