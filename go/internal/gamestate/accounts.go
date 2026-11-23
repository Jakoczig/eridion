package gamestate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Jakoczig/eridion/go/internal/catalog"
	"github.com/Jakoczig/eridion/go/internal/history"
	"github.com/Jakoczig/eridion/go/internal/privatefs"
	"os"
	"strconv"
	"time"
)

// AccountSettings holds the persisted commander preferences. Default is the
// configured single-mode commander; Selected is the separate, temporary
// multi-mode choice. Neither follows journal import order.
type AccountSettings struct {
	Version  int    `json:"version,omitempty"`
	Multiple bool   `json:"multiple"`
	Selected string `json:"selected"`
	Default  string `json:"defaultFID,omitempty"`
}

const (
	accountSettingsFile     = "accounts-v1.json"
	accountRecoveryFile     = "accounts-v1.recovery.json"
	accountSettingsVersion  = 2
	maxAccountSettingsBytes = 4096
)

func validAccountFID(fid string) bool { return fid == "" || (len(fid) <= 128 && safeText(fid)) }

// LoadAccountSettings strictly decodes the versioned preference file. A
// missing file is a valid first run. A damaged or unsupported file returns
// an error so callers keep it for recovery instead of silently choosing a
// different account. A prior explicit selection migrates to the default.
func LoadAccountSettings(dir string) (AccountSettings, error) {
	var out AccountSettings
	if dir == "" {
		return out, nil
	}
	raw, err := privatefs.Read(dir, accountSettingsFile)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if len(raw) > maxAccountSettingsBytes {
		return AccountSettings{}, errors.New("Account preferences exceed limit")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(&out); err != nil || d.More() {
		return AccountSettings{}, errors.New("Account preferences are damaged")
	}
	if out.Version < 0 || out.Version > accountSettingsVersion || !validAccountFID(out.Selected) || !validAccountFID(out.Default) {
		return AccountSettings{}, errors.New("Account preferences are unsupported")
	}
	if out.Default == "" && out.Selected != "" {
		out.Default = out.Selected
	}
	return out, nil
}

// loadAccountsForService keeps an unreadable preference file aside for
// recovery (without overwriting an earlier preserved copy) and starts with
// no default, so the launcher asks once instead of guessing.
func loadAccountsForService(dir string) (AccountSettings, string) {
	settings, err := LoadAccountSettings(dir)
	if err == nil {
		return settings, ""
	}
	if raw, e := privatefs.Read(dir, accountSettingsFile); e == nil {
		if _, e = privatefs.Read(dir, accountRecoveryFile); os.IsNotExist(e) {
			_ = privatefs.Write(dir, accountRecoveryFile, raw)
		}
	}
	return AccountSettings{}, err.Error()
}

func (s *Service) displayFIDLocked() string {
	if s.accounts.Multiple {
		return s.accounts.Selected
	}
	if s.accounts.Default != "" && s.defaultInvalid == "" {
		return s.accounts.Default
	}
	return s.active
}

// PinnedAccount is the commander whose provider namespace and remote account
// binding must stay fixed regardless of the journal-active commander.
func (s *Service) PinnedAccount() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hostAccountLocked()
}

// hostAccountLocked is the host-authorized commander: the launcher selection
// in multi mode, otherwise the resolved default. Empty while unresolved.
func (s *Service) hostAccountLocked() string {
	if s.accounts.Multiple {
		return s.accounts.Selected
	}
	if s.defaultInvalid != "" {
		return ""
	}
	return s.accounts.Default
}

// DefaultRequired reports a one-time launcher setup: several verified
// commanders are known, or the stored preference could not be read, and no
// default has been chosen yet.
func (s *Service) DefaultRequired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.defaultRequiredLocked()
}
func (s *Service) defaultRequiredLocked() bool {
	if s.defaultInvalid != "" {
		return true
	}
	return s.accounts.Default == "" && (s.accountsError != "" || len(s.order) > 1)
}

// AccountRecovery describes a preference file that was preserved for
// recovery, or "" when the stored preferences were read normally.
func (s *Service) AccountRecovery() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.accountsError != "" {
		return s.accountsError
	}
	return s.defaultInvalid
}

// validateDefaultLocked runs once history is complete. A stored default, or a
// legacy selection migrated to it, that names no verified commander is not
// used: the launcher asks once and providers stay blocked. It is never
// replaced by another commander automatically.
func (s *Service) validateDefaultLocked() {
	if s.accounts.Default == "" {
		s.defaultInvalid = ""
		return
	}
	_, known := s.commanders[s.accounts.Default]
	was := s.defaultInvalid
	if known {
		s.defaultInvalid = ""
	} else {
		s.defaultInvalid = "The default commander is not among the verified local commanders. Choose the default commander in the launcher."
	}
	if was != s.defaultInvalid && !s.accounts.Multiple && s.opts.ProviderAccount != nil {
		s.opts.ProviderAccount(s.hostAccountLocked())
	}
}

// adoptDefaultLocked records the sole verified commander as default when no
// default exists. Several cached identities stay ambiguous for the launcher;
// damaged preferences are never replaced automatically.
func (s *Service) adoptDefaultLocked() {
	if s.accounts.Default != "" || s.accountsError != "" || s.summary.ImportInProgress || len(s.order) != 1 || !validAccountFID(s.order[0]) || s.order[0] == "" {
		return
	}
	next := s.accounts
	next.Default = s.order[0]
	if s.persistAccountsLocked(next) == nil {
		s.accounts = next
		if !next.Multiple && s.opts.ProviderAccount != nil {
			s.opts.ProviderAccount(next.Default)
		}
	}
}

func (s *Service) persistAccountsLocked(next AccountSettings) error {
	next.Version = accountSettingsVersion
	if s.opts.StorageDir == "" {
		return nil
	}
	raw, _ := json.Marshal(next)
	return privatefs.Write(s.opts.StorageDir, accountSettingsFile, raw)
}

func (s *Service) selectedSummaryLocked() Summary {
	v := s.summary
	if fid := s.displayFIDLocked(); fid != s.active {
		v.CurrentSystem = s.commanders[fid].system
		v.LastEventType = ""
		v.LastEventAt = ""
	}
	return v
}
func (s *Service) Accounts() AccountSettings { s.mu.Lock(); defer s.mu.Unlock(); return s.accounts }

// ShipIdentity is the single resolved main-ship identity shown by every
// screen (Commander, header/statistics, account cards, paired displays).
type ShipIdentity struct {
	Name       string `json:"shipName"`
	Model      string `json:"model"`
	ModelLabel string `json:"shipType"`
	Source     string `json:"source"`
}

// shipIdentityLocked applies D0 to the main ship: a fresh validated Frontier
// current ship first, then the journal ship view, then the durable witnessed
// rename for that exact commander/ship ID when it is not older than the
// view's Loadout (covering reducer caches that predate rename support). Suits and vehicles never appear here because the
// ship reducer keeps the real main ship.
func (s *Service) shipIdentityLocked(ctx context.Context, fid string, facts []history.EffectiveFact) ShipIdentity {
	v := s.views.ship.View(fid)
	out := ShipIdentity{Name: v.ShipName, Model: v.ShipType, Source: "journal"}
	if s.store != nil && fid != "" {
		var name, at string
		if e := s.store.WitnessedShipName(ctx, fid, s.views.ship.LastKnownIdentity(fid), &name, &at); e == nil && name != "" {
			observed, e := time.Parse(time.RFC3339Nano, at)
			loaded, _ := time.Parse(time.RFC3339Nano, v.LoadoutAt)
			if e == nil && !observed.Before(loaded) {
				out.Name = name
			}
		}
	}
	if ship, ok := history.ResolveCurrentShip(facts); ok {
		out.Source = ship.Source
		if ship.Name != "" {
			out.Name = ship.Name
		}
		if ship.Model != "" {
			out.Model = ship.Model
		}
	}
	out.ModelLabel = catalog.ShipDisplayName(out.Model)
	return out
}

func (s *Service) factsLocked(ctx context.Context, fid string) []history.EffectiveFact {
	if s.store == nil || fid == "" {
		return nil
	}
	facts, _ := s.store.EffectiveFacts(ctx, fid, "commander", time.Now())
	return facts
}

// OwnerShipIdentity returns the resolved ship identity of the displayed
// commander. Exact commander/ship ID binding is mandatory.
func (s *Service) OwnerShipIdentity() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.summary.ImportInProgress {
		return map[string]string{"shipName": "", "shipType": ""}
	}
	fid := s.displayFIDLocked()
	ctx := context.Background()
	id := s.shipIdentityLocked(ctx, fid, s.factsLocked(ctx, fid))
	return map[string]string{"shipName": id.Name, "shipType": id.ModelLabel, "source": id.Source}
}

// SetAccounts changes multi-mode and its temporary selection. The persisted
// default is independent and preserved.
func (s *Service) SetAccounts(multiple bool, fid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !multiple {
		fid = ""
	} else {
		if fid == "" {
			fid = s.displayFIDLocked()
		}
		if _, ok := s.commanders[fid]; !ok || fid == "" {
			return errors.New("Choose a commander witnessed in verified local journals")
		}
	}
	next := s.accounts
	next.Multiple, next.Selected = multiple, fid
	return s.applyAccountsLocked(next)
}

// SetDefaultAccount persists the single-mode default commander. Only a
// commander witnessed in verified local journals can be chosen.
func (s *Service) SetDefaultAccount(fid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.commanders[fid]; !ok || fid == "" || !validAccountFID(fid) {
		return errors.New("Choose a commander witnessed in verified local journals")
	}
	next := s.accounts
	next.Default = fid
	if err := s.applyAccountsLocked(next); err != nil {
		return err
	}
	s.accountsError, s.defaultInvalid = "", ""
	if !s.accounts.Multiple && s.opts.ProviderAccount != nil {
		s.opts.ProviderAccount(fid)
	}
	return nil
}

// RestoreAccounts reinstates previously read settings exactly, for rollback
// after a failed multi-step account change.
func (s *Service) RestoreAccounts(prev AccountSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyAccountsLocked(prev)
}

func (s *Service) applyAccountsLocked(next AccountSettings) error {
	if e := s.persistAccountsLocked(next); e != nil {
		return errors.New("Cannot persist account selection; previous selection retained")
	}
	next.Version = accountSettingsVersion
	s.accounts = next
	s.cachedProjection = nil
	s.views.dirty = true
	s.publishLocked()
	return nil
}

type AccountCard struct {
	FID      string `json:"fid"`
	Name     string `json:"name"`
	ShipName string `json:"shipName"`
	ShipType string `json:"shipType"`
	System   string `json:"system"`
	Active   bool   `json:"active"`
	Default  bool   `json:"default"`
}

func (s *Service) AccountCards() []AccountCard {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AccountCard{}
	names := map[string]string{}
	if s.store != nil {
		if people, e := s.store.Commanders(); e == nil {
			for _, p := range people {
				names[p.FID] = p.Name
			}
		}
	}
	ctx := context.Background()
	for _, fid := range s.order {
		ship := s.shipIdentityLocked(ctx, fid, s.factsLocked(ctx, fid))
		out = append(out, AccountCard{fid, names[fid], ship.Name, ship.ModelLabel, s.commanders[fid].system, fid == s.displayFIDLocked(), fid == s.accounts.Default})
	}
	return out
}
func (s *Service) CommanderOverview(ctx context.Context) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fid := s.displayFIDLocked()
	out := map[string]any{"fid": fid, "system": s.commanders[fid].system, "journalMismatch": fid != s.active}
	if s.store == nil {
		ship := s.shipIdentityLocked(ctx, fid, nil)
		out["shipName"], out["shipType"], out["identitySource"] = ship.Name, ship.ModelLabel, ship.Source
		return out, nil
	}
	people, e := s.store.Commanders()
	if e != nil {
		return nil, e
	}
	for _, p := range people {
		if p.FID == fid {
			out["name"] = p.Name
		}
	}
	facts, e := s.store.EffectiveFacts(ctx, fid, "commander", time.Now())
	if e != nil {
		return nil, e
	}
	out["facts"] = facts
	// D0: ranks and current ship identity come from the same resolver as every
	// other commander fact; the journal view is only the fallback.
	journalRanks := []history.JournalRank{}
	for _, r := range s.views.activity.View(fid).Commander.Ranks {
		journalRanks = append(journalRanks, history.JournalRank{Category: r.Category, Rank: decimalText(r.Rank), Progress: decimalText(r.Progress)})
	}
	out["ranks"] = history.MergeJournalRanks(history.ResolveRanks(facts), journalRanks)
	ship := s.shipIdentityLocked(ctx, fid, facts)
	out["shipName"], out["shipType"], out["identitySource"] = ship.Name, ship.ModelLabel, ship.Source
	if current, ok := history.ResolveCurrentShip(facts); ok {
		out["identityFetchedAt"] = current.FetchedAt
		if current.SystemName != "" {
			out["system"] = current.SystemName
		}
	}
	fleet, e := s.store.ReconciledFleetCount(ctx, fid, facts)
	if e != nil {
		return nil, e
	}
	out["fleetCount"] = fleet
	power, e := s.store.Operations(ctx, fid, history.OperationQuery{Kind: "powerplay", Limit: 10})
	if e == nil {
		out["powerplay"] = power.Current
	}
	squad, e := s.store.Operations(ctx, fid, history.OperationQuery{Kind: "squadron", Limit: 10})
	if e == nil {
		out["squadron"] = squad.Current
	}
	return out, nil
}

// decimalText renders a reducer integer value as exact decimal text.
func decimalText(v *float64) *string {
	if v == nil {
		return nil
	}
	text := strconv.FormatFloat(*v, 'f', -1, 64)
	return &text
}
