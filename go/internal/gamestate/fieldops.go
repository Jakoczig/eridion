package gamestate

import (
	"context"
	"encoding/json"
	"github.com/Jakoczig/eridion/go/internal/explorestate"
	"github.com/Jakoczig/eridion/go/internal/fieldops"
	"github.com/Jakoczig/eridion/go/internal/history"
	"time"
)

func (s *Service) FieldCommander() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.displayFIDLocked()
}
func (s *Service) captureBioLocked() {
	if s.store == nil || len(s.bioStatus) == 0 || s.active == "" || s.stopped {
		return
	}
	if !s.store.BuiltinEnabled("bioscan") {
		return
	}
	var v struct{ BodyName string }
	if json.Unmarshal(s.bioStatus, &v) != nil {
		return
	}
	state := s.commanders[s.active]
	if state.systemAddress == "" {
		return
	}
	survey := s.views.exploration.View(s.active, state.systemAddress)
	for _, b := range survey.Bodies {
		if b.Name == v.BodyName && b.BodyID != "" {
			if err := s.store.RecordBioPosition(s.active, state.systemAddress, b.BodyID, b.Name, s.bioStatus, time.Now(), s.activeSource); err != nil {
				s.bioStatus = nil
			}
			return
		}
	}
}

type FieldRouteState struct {
	Progress        history.ItineraryProgress
	Enabled         bool
	Total           int
	Current, Target string
	Evidence        history.NavigationEvidence
	Plans           []history.Itinerary
}

func (s *Service) FieldProgress(owner bool) (FieldRouteState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out FieldRouteState
	if s.store == nil {
		return out, nil
	}
	out.Enabled = s.store.BuiltinEnabled("itinerary")
	var e error
	out.Progress, e = s.store.ItineraryProgress(s.displayFIDLocked())
	if e != nil {
		return out, e
	}
	out.Total, e = s.store.ItinerarySize(out.Progress.Plan.ID)
	if e != nil {
		return out, e
	}
	out.Current = s.selectedSummaryLocked().CurrentSystem
	out.Target = s.projectionLocked().Navigation.TargetSystem
	out.Evidence, e = s.store.FieldNavigationEvidence(s.displayFIDLocked())
	if e != nil {
		return out, e
	}
	if owner {
		out.Plans, e = s.store.Itineraries(s.displayFIDLocked())
	}
	return out, e
}

// bioContext is the state a BioScan read is computed for. A reply is only
// returned while the service still shows this context (Task 18).
type bioContext struct {
	fid, system, body string
	survey            explorestate.BodyView
	allBodiesFound    bool
	live              bool
	activeSource      string
	status            []byte
}

func (s *Service) bioContextLocked(body string) (bioContext, bool) {
	fid := s.displayFIDLocked()
	state := s.commanders[fid]
	c := bioContext{fid: fid, system: state.systemAddress, body: body}
	if state.systemAddress == "" {
		return c, false
	}
	survey := s.views.exploration.View(fid, state.systemAddress)
	c.allBodiesFound = survey.AllBodiesFound
	for _, b := range survey.Bodies {
		if b.BodyID == body {
			c.survey = b
			c.live = fid == s.active && !s.stopped && s.summary.SourceState == "live"
			c.activeSource = s.activeSource
			c.status = append([]byte(nil), s.bioStatus...)
			return c, true
		}
	}
	return c, false
}

// FieldBio returns the BioScan view for a body of the displayed commander's
// current system. Private normalized scans and commander identity never
// cross the viewer boundary. The service mutex is held only to snapshot the
// context and to revalidate it: database reads and candidate inference run
// outside it, and identical concurrent inferences share one computation.
func (s *Service) FieldBio(body string, owner bool) (map[string]any, error) {
	out := map[string]any{"version": fieldops.BioRuleVersion(), "candidates": []fieldops.Candidate{}, "samples": []history.BioSample{}, "body": body}
	s.mu.Lock()
	store := s.store
	c, found := s.bioContextLocked(body)
	s.mu.Unlock()
	if store == nil {
		return out, nil
	}
	if !store.BuiltinEnabled("bioscan") {
		out["notice"] = "BioScan module disabled; stored observations retained"
		return out, nil
	}
	if !found {
		return out, nil
	}
	v, err := store.BioBody(context.Background(), c.fid, c.system, body)
	if err != nil {
		return out, err
	}
	// Stable genus symbols only (Task 6); localized labels never identify
	// a genus.
	genera := c.survey.GenusIDs
	if len(genera) == 0 {
		genera, _ = v["genera"].([]string)
	}
	if c.survey.BioSignals != nil && *c.survey.BioSignals > 0 {
		scan, _ := v["environment"].(map[string]any)
		env, _ := v["system"].(fieldops.Environment)
		env.Body = scan
		env.AllBodiesFound = c.allBodiesFound
		v["candidates"] = s.bioCandidates(c, env, genera)
	}
	delete(v, "scan")
	delete(v, "environment")
	delete(v, "genera")
	delete(v, "system")
	v["body"] = body
	v["confirmed"] = c.survey.Organisms
	v["signals"] = c.survey.BioSignals
	v["bodyName"] = c.survey.Name
	samples := v["samples"].([]history.BioSample)
	if c.live && len(samples) > 0 && (samples[0].Stage == "Log" || samples[0].Stage == "Sample") && samples[0].Session != "" && samples[0].Session == store.BioSession(c.fid, c.activeSource) && samples[0].Event == store.ActiveOrganicEvent(c.fid, c.activeSource) {
		if current, ok := fieldops.StatusPosition(c.status, c.survey.Name, time.Now(), time.Now()); ok {
			points := []fieldops.Position{}
			complete := false
			expected := 0
			for _, sample := range samples {
				if sample.Session != samples[0].Session || sample.Genus != samples[0].Genus || sample.Species != samples[0].Species || sample.Variant != samples[0].Variant {
					continue
				}
				if sample.Stage == "Analyse" {
					break
				}
				expected++
				if sample.Position != nil {
					points = append(points, *sample.Position)
				}
				if sample.Stage == "Log" {
					complete = len(points) > 0 && len(points) == expected
					break
				}
			}
			v["liveSpacing"] = fieldops.LiveSpacing(current, points, fieldops.MinimumDistance(samples[0].Genus), complete)
		}
	}
	if !owner {
		for i := range samples {
			samples[i].Position = nil
		}
		v["samples"] = samples
	}
	// Revalidate: if the displayed commander or system changed while this
	// ran, the reply belongs to a context that is no longer shown.
	s.mu.Lock()
	now, still := s.bioContextLocked(body)
	s.mu.Unlock()
	if !still || now.fid != c.fid || now.system != c.system {
		return map[string]any{"version": fieldops.BioRuleVersion(), "candidates": []fieldops.Candidate{}, "samples": []history.BioSample{}, "body": body, "notice": "Context changed while BioScan was computed; refreshed"}, nil
	}
	return v, nil
}
