package paritystate

import "time"

// Exosuit equipment is known, account-scoped history: it survives shutdown,
// source gaps and new sessions of the same commander, and is only replaced by
// newer explicit suit/loadout events. Grades are never inferred from a
// "_classN" suffix; only an explicit Class (UpgradeSuit) sets one.
const (
	maxSuitLoadouts = 64
	maxSuits        = 64
	maxSuitModules  = 16
	maxSuitMods     = 16
)

type ExosuitView struct {
	State      string        `json:"state"`
	ObservedAt string        `json:"observedAt"`
	Controlled string        `json:"controlled"`
	CurrentID  string        `json:"currentLoadoutId"`
	Current    *SuitLoadout  `json:"current"`
	Loadouts   []SuitLoadout `json:"loadouts"`
	Suits      []SuitRecord  `json:"suits"`
	// Live is attached by the host only while a verified, current-account
	// Status reports the commander on foot.
	Live *OnFootStatus `json:"live"`
}
type SuitLoadout struct {
	LoadoutID string       `json:"loadoutId"`
	Name      string       `json:"name"`
	SuitID    string       `json:"suitId"`
	Suit      string       `json:"suit"`
	SuitLabel string       `json:"suitLabel"`
	Grade     *int         `json:"grade"`
	Mods      []string     `json:"mods"`
	Modules   []SuitModule `json:"modules"`
}
type SuitModule struct {
	Slot     string   `json:"slot"`
	ModuleID string   `json:"moduleId"`
	Module   string   `json:"module"`
	Label    string   `json:"label"`
	Class    *int     `json:"class"`
	Mods     []string `json:"mods"`
}
type SuitRecord struct {
	SuitID string   `json:"suitId"`
	Suit   string   `json:"suit"`
	Label  string   `json:"label"`
	Grade  *int     `json:"grade"`
	Mods   []string `json:"mods"`
}
type OnFootStatus struct {
	Health              *float64 `json:"health"`
	Oxygen              *float64 `json:"oxygen"`
	TemperatureK        *float64 `json:"temperatureK"`
	GravityG            *float64 `json:"gravityG"`
	SelectedWeapon      string   `json:"selectedWeapon"`
	SelectedWeaponLabel string   `json:"selectedWeaponLabel"`
	BodyName            string   `json:"bodyName"`
	LowHealth           bool     `json:"lowHealth"`
	LowOxygen           bool     `json:"lowOxygen"`
	BreathableAtmos     bool     `json:"breathableAtmosphere"`
}

func blankExosuit() ExosuitView {
	return ExosuitView{State: "unknown", Loadouts: []SuitLoadout{}, Suits: []SuitRecord{}}
}

func cloneStrings(v []string) []string {
	if v == nil {
		return nil
	}
	return append([]string{}, v...)
}

func cloneSuitLoadout(l SuitLoadout) SuitLoadout {
	l.Grade = copyPtr(l.Grade)
	l.Mods = cloneStrings(l.Mods)
	if l.Modules != nil {
		l.Modules = append([]SuitModule{}, l.Modules...)
		for i := range l.Modules {
			l.Modules[i].Class = copyPtr(l.Modules[i].Class)
			l.Modules[i].Mods = cloneStrings(l.Modules[i].Mods)
		}
	}
	return l
}

func cloneOnFoot(v *OnFootStatus) *OnFootStatus {
	if v == nil {
		return nil
	}
	live := *v
	live.Health, live.Oxygen, live.TemperatureK, live.GravityG = copyPtr(live.Health), copyPtr(live.Oxygen), copyPtr(live.TemperatureK), copyPtr(live.GravityG)
	return &live
}

// cloneExosuit detaches every mutable field and preserves nil versus empty
// collections exactly.
func cloneExosuit(v ExosuitView) ExosuitView {
	out := v
	if v.Loadouts != nil {
		out.Loadouts = make([]SuitLoadout, len(v.Loadouts))
		for i := range v.Loadouts {
			out.Loadouts[i] = cloneSuitLoadout(v.Loadouts[i])
		}
	}
	if v.Suits != nil {
		out.Suits = append([]SuitRecord{}, v.Suits...)
		for i := range out.Suits {
			out.Suits[i].Grade = copyPtr(out.Suits[i].Grade)
			out.Suits[i].Mods = cloneStrings(out.Suits[i].Mods)
		}
	}
	if v.Current != nil {
		current := cloneSuitLoadout(*v.Current)
		out.Current = &current
	}
	out.Live = cloneOnFoot(v.Live)
	return out
}

// CloneExosuit returns a detached copy of v.
func CloneExosuit(v ExosuitView) ExosuitView { return cloneExosuit(v) }

// Exosuit returns a detached view with the current loadout resolved and the
// controlled entity from the ship reducer. Live telemetry is attached by the
// host from the verified Status view.
func (m *Model) Exosuit(key string) ExosuitView {
	s := m.states[key]
	if s == nil {
		return blankExosuit()
	}
	out := cloneExosuit(s.v.Exosuit)
	if out.State == "" {
		out = blankExosuit()
	}
	out.Live, out.Current = nil, nil
	out.Controlled = s.v.Ship.Controlled
	for i := range out.Loadouts {
		l := &out.Loadouts[i]
		if suit := out.suit(l.SuitID); suit != nil && suit.Grade != nil {
			l.Grade = copyPtr(suit.Grade)
		}
		if l.LoadoutID == out.CurrentID && out.CurrentID != "" {
			current := cloneSuitLoadout(*l)
			out.Current = &current
		}
	}
	return out
}

func stringList(v any, limit int) []string {
	out := []string{}
	items, _ := v.([]any)
	for _, item := range items {
		if s := text(item); s != "" && len(out) < limit {
			out = append(out, s)
		}
	}
	return out
}

func suitModule(d map[string]any) (SuitModule, bool) {
	m := SuitModule{Slot: text(d["SlotName"]), ModuleID: ident(d["SuitModuleID"]), Module: text(d["ModuleName"]), Label: text(d["ModuleName_Localised"]), Class: integer(d["Class"], 5), Mods: stringList(d["WeaponMods"], maxSuitMods)}
	return m, m.Slot != ""
}

func suitLoadout(d map[string]any) (SuitLoadout, bool) {
	l := SuitLoadout{LoadoutID: ident(d["LoadoutID"]), Name: text(d["LoadoutName"]), SuitID: ident(d["SuitID"]), Suit: text(d["SuitName"]), SuitLabel: text(d["SuitName_Localised"]), Mods: stringList(d["SuitMods"], maxSuitMods), Modules: []SuitModule{}}
	if l.LoadoutID == "" || l.SuitID == "" {
		return l, false
	}
	items, _ := d["Modules"].([]any)
	for _, item := range items {
		x, _ := item.(map[string]any)
		if mod, ok := suitModule(x); ok && len(l.Modules) < maxSuitModules {
			l.Modules = append(l.Modules, mod)
		}
	}
	return l, true
}

func (v *ExosuitView) loadout(id string) *SuitLoadout {
	for i := range v.Loadouts {
		if v.Loadouts[i].LoadoutID == id {
			return &v.Loadouts[i]
		}
	}
	return nil
}

func (v *ExosuitView) suit(id string) *SuitRecord {
	for i := range v.Suits {
		if v.Suits[i].SuitID == id {
			return &v.Suits[i]
		}
	}
	return nil
}

func (v *ExosuitView) upsertLoadout(l SuitLoadout) {
	if old := v.loadout(l.LoadoutID); old != nil {
		*old = l
		return
	}
	if len(v.Loadouts) >= maxSuitLoadouts {
		v.State = "partial"
		return
	}
	v.Loadouts = append(v.Loadouts, l)
}

// applyExosuit reduces explicit Odyssey suit/loadout events and reports
// whether the event was one of them.
func applyExosuit(v *ExosuitView, kind string, at time.Time, d map[string]any) bool {
	switch kind {
	case "SuitLoadout", "SwitchSuitLoadout", "CreateSuitLoadout", "RenameSuitLoadout", "DeleteSuitLoadout", "LoadoutEquipModule", "LoadoutRemoveModule", "BuySuit", "SellSuit", "UpgradeSuit", "UpgradeWeapon":
	default:
		return false
	}
	if v.State == "" {
		*v = blankExosuit()
	}
	if v.State == "unknown" {
		v.State = "known"
	}
	v.ObservedAt = at.UTC().Format(time.RFC3339Nano)
	partial := func() { v.State = "partial" }
	switch kind {
	case "SuitLoadout", "SwitchSuitLoadout", "CreateSuitLoadout":
		l, ok := suitLoadout(d)
		if !ok {
			partial()
			return true
		}
		v.upsertLoadout(l)
		if kind != "CreateSuitLoadout" {
			v.CurrentID = l.LoadoutID
		}
		if v.suit(l.SuitID) == nil && len(v.Suits) < maxSuits {
			v.Suits = append(v.Suits, SuitRecord{SuitID: l.SuitID, Suit: l.Suit, Label: l.SuitLabel, Mods: append([]string{}, l.Mods...)})
		}
	case "RenameSuitLoadout":
		if l := v.loadout(ident(d["LoadoutID"])); l != nil && text(d["LoadoutName"]) != "" {
			l.Name = text(d["LoadoutName"])
		} else {
			partial()
		}
	case "DeleteSuitLoadout":
		id := ident(d["LoadoutID"])
		for i := range v.Loadouts {
			if v.Loadouts[i].LoadoutID == id {
				v.Loadouts = append(v.Loadouts[:i], v.Loadouts[i+1:]...)
				break
			}
		}
		if v.CurrentID == id {
			v.CurrentID = ""
		}
	case "LoadoutEquipModule", "LoadoutRemoveModule":
		l := v.loadout(ident(d["LoadoutID"]))
		mod, ok := suitModule(d)
		if l == nil || !ok {
			partial()
			return true
		}
		kept := []SuitModule{}
		for _, existing := range l.Modules {
			if existing.Slot != mod.Slot {
				kept = append(kept, existing)
			}
		}
		l.Modules = kept
		if kind == "LoadoutEquipModule" && len(l.Modules) < maxSuitModules {
			l.Modules = append(l.Modules, mod)
		}
	case "BuySuit":
		id := ident(d["SuitID"])
		if id != "" && v.suit(id) == nil && len(v.Suits) < maxSuits {
			v.Suits = append(v.Suits, SuitRecord{SuitID: id, Suit: text(d["Name"]), Label: text(d["Name_Localised"]), Mods: stringList(d["SuitMods"], maxSuitMods)})
		}
	case "SellSuit":
		id := ident(d["SuitID"])
		for i := range v.Suits {
			if v.Suits[i].SuitID == id {
				v.Suits = append(v.Suits[:i], v.Suits[i+1:]...)
				break
			}
		}
	case "UpgradeSuit":
		id, class := ident(d["SuitID"]), integer(d["Class"], 5)
		s := v.suit(id)
		if s == nil && id != "" && len(v.Suits) < maxSuits {
			v.Suits = append(v.Suits, SuitRecord{SuitID: id, Suit: text(d["Name"]), Mods: []string{}})
			s = &v.Suits[len(v.Suits)-1]
		}
		if s == nil || class == nil {
			partial()
			return true
		}
		s.Grade = class
	case "UpgradeWeapon":
		id, class := ident(d["SuitModuleID"]), integer(d["Class"], 5)
		if id == "" || class == nil {
			partial()
			return true
		}
		for i := range v.Loadouts {
			for j := range v.Loadouts[i].Modules {
				if v.Loadouts[i].Modules[j].ModuleID == id {
					c := *class
					v.Loadouts[i].Modules[j].Class = &c
				}
			}
		}
	}
	return true
}
