package gamestate

import (
	"context"
	"github.com/Jakoczig/eridion/go/internal/activitystate"
	"github.com/Jakoczig/eridion/go/internal/explorestate"
	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/navcontext"
	"github.com/Jakoczig/eridion/go/internal/navstate"
	"github.com/Jakoczig/eridion/go/internal/paritystate"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
	"github.com/Jakoczig/eridion/go/internal/shipstate"
	"github.com/Jakoczig/eridion/go/internal/sidecar"
	"sync"
)

// Projections is an internal, immutable, coalesced notification. Transport sends
// only the separately typed allowlisted DTOs, never this service container.
type Projections struct {
	// Revisions identify each domain's published value (not sent to clients).
	Revisions   Revisions
	Workspace   *WorkspaceView
	Ship        shipstate.View
	Source      Summary
	Navigation  navstate.NavigationView
	Exploration explorestate.ExplorationView
}
type WorkspaceView struct {
	Parity   paritystate.View   `json:"parity"`
	Activity activitystate.View `json:"activity"`
	// Exosuit holds known suit/loadout equipment (available offline) and, only
	// while the verified current-account Status reports the commander on
	// foot, live suit telemetry.
	Exosuit paritystate.ExosuitView `json:"exosuit"`
}

type projectionState struct {
	parity      *paritystate.Model
	activity    *activitystate.Model
	ship        *shipstate.Model
	navigation  *navstate.Model
	exploration *explorestate.Model
	subscribers map[chan Projections]struct{}
	last        Projections
	hasLast     bool
	// dirty: a change that may affect every domain. partial: changes
	// confined to these domains. deferred: domains changed while nobody
	// demanded them (their last value is outdated).
	dirty             bool
	partial, deferred uint8
	demand            uint8
	rev               Revisions
	lastKey           projectionKey
}

func newProjectionState() projectionState {
	return projectionState{parity: paritystate.NewModel(), activity: activitystate.NewModel(), ship: shipstate.NewModel(), navigation: navstate.NewModel(), exploration: explorestate.NewModel(), subscribers: map[chan Projections]struct{}{}, dirty: true, demand: DomainAll}
}
func (s *Service) projectionLocked() Projections {
	perfstats.Add("projection.build", 1)
	fid := s.displayFIDLocked()
	if s.cachedProjection != nil && s.summary.ImportInProgress && fid != "" && fid == s.cachedProjectionFID {
		p := cloneProjection(*s.cachedProjection)
		p.Source = s.selectedSummaryLocked()
		p.Ship.Evidence = "cached"
		p.Navigation.RouteState = "cached"
		p.Exploration.SurveyState = "cached"
		if p.Workspace != nil {
			p.Workspace.Parity.Evidence = "cached"
			p.Workspace.Parity.Status.State = "stale_source"
		}
		return p
	}

	return Projections{Source: s.selectedSummaryLocked(), Ship: s.shipDomainLocked(fid), Navigation: s.navigationDomainLocked(fid), Exploration: s.explorationDomainLocked(fid), Workspace: s.workspaceDomainLocked(fid)}
}

// staleSource reports a source state whose projections are marked stale.
func (s *Service) staleSourceLocked() bool {
	return s.summary.SourceState != "live" && s.summary.SourceState != "replaying" && s.summary.SourceState != "partial" && s.summary.SourceState != "parse_warning"
}

// The domain builders below are the parts of the projection; each applies
// its own replaying/stale/cached presentation exactly as the combined
// projection did (G3K PERF-05: a change in one domain rebuilds only it).

func (s *Service) shipDomainLocked(fid string) shipstate.View {
	perfstats.Add("domain.build.ship", 1)
	v := s.views.ship.View(fid)
	if s.summary.ImportInProgress {
		v = shipstate.View{Evidence: "replaying"}
	} else if s.staleSourceLocked() {
		v.Evidence = "stale_source"
	}
	if s.summary.Cached {
		v.Evidence = "cached"
	}
	return v
}

func (s *Service) navigationDomainLocked(fid string) navstate.NavigationView {
	perfstats.Add("domain.build.navigation", 1)
	v := s.views.navigation.View(fid)
	if s.summary.ImportInProgress {
		v = navstate.NavigationView{RouteState: "replaying", TargetEvidence: "unknown", Hops: []navstate.Hop{}}
	} else if s.staleSourceLocked() && !s.summary.Cached && (len(v.Hops) > 0 || v.TargetSystem != "") {
		v.Hops = []navstate.Hop{}
		v.DestinationSystem = ""
		v.TargetSystem = ""
		v.RemainingJumps = nil
		v.ReportedRemainingJumps = nil
		v.RouteState = "stale"
		v.TargetEvidence = "unknown"
	}
	if s.summary.Cached && (v.RouteState == "verified_last_known" || v.RouteState == "verified") {
		v.RouteState = "cached"
		v.TargetEvidence = "cached"
	}
	return v
}

func (s *Service) explorationDomainLocked(fid string) explorestate.ExplorationView {
	perfstats.Add("domain.build.exploration", 1)
	address := s.unknown.systemAddress
	if fid != "" {
		address = s.commanders[fid].systemAddress
	}
	v := s.views.exploration.View(fid, address)
	if s.summary.ImportInProgress {
		v = explorestate.ExplorationView{SurveyState: "replaying", Bodies: []explorestate.BodyView{}}
	} else if s.staleSourceLocked() && len(v.Bodies) > 0 {
		v.SurveyState = "stale_source"
	}
	if s.summary.Cached && v.SystemName != "" {
		v.SurveyState = "cached"
	}
	return v
}

func (s *Service) workspaceDomainLocked(fid string) *WorkspaceView {
	if !s.opts.EnableParity {
		return nil
	}
	perfstats.Add("domain.build.workspace", 1)
	w := WorkspaceView{Parity: s.views.parity.View(fid), Activity: s.views.activity.View(fid), Exosuit: s.views.parity.Exosuit(fid)}
	if refs := s.frontierModuleRefsLocked(fid); len(refs) > 0 && !s.summary.ImportInProgress {
		w.Parity.Modules = paritystate.ReconcileModules(w.Parity.Modules, refs)
	}
	if s.summary.ImportInProgress {
		w = WorkspaceView{Parity: paritystate.View{Evidence: "replaying"}, Activity: activitystate.View{Evidence: "replaying"}, Exosuit: s.views.parity.Exosuit(fid)}
	} else if s.summary.SourceState != "live" && s.summary.SourceState != "partial" && s.summary.SourceState != "parse_warning" {
		w.Parity.Evidence = "stale_source"
		w.Parity.Status = paritystate.StatusView{State: "stale_source"}
		w.Activity.Evidence = "stale_source"
	}
	if fid != s.active {
		w.Parity.Status = paritystate.StatusView{State: "different_account"}
	}
	// Live suit telemetry requires the final, account-checked Status: a
	// stale source, another account, embarking or shutdown remove it while
	// known equipment stays available.
	w.Exosuit.Live = nil
	if st := w.Parity.Status; st.State == "known" && st.OnFoot != nil && !s.summary.Cached {
		w.Exosuit.Live = paritystate.CloneExosuit(paritystate.ExosuitView{Live: st.OnFoot}).Live
	}
	if s.summary.Cached {
		w.Parity.Evidence = "cached"
		w.Activity.Evidence = "cached"
	}
	return &w
}

func (s *Service) Navigation() navstate.NavigationView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedImportLocked() {
		return s.projectionLocked().Navigation
	}
	if s.domainFreshLocked(DomainNavigation) {
		perfstats.Add("getter.reuse.navigation", 1)
		return cloneProjection(Projections{Navigation: s.views.last.Navigation}).Navigation
	}
	return s.navigationDomainLocked(s.displayFIDLocked())
}
func (s *Service) Exploration() explorestate.ExplorationView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedImportLocked() {
		return s.projectionLocked().Exploration
	}
	if s.domainFreshLocked(DomainExploration) {
		perfstats.Add("getter.reuse.exploration", 1)
		return cloneProjection(Projections{Exploration: s.views.last.Exploration}).Exploration
	}
	return s.explorationDomainLocked(s.displayFIDLocked())
}
func (s *Service) applyProjectionsLocked(e journal.Event, phase journal.Phase) {
	batch := s.startup.Observe(e)
	s.views.ship.ApplyEvent(s.active, e)
	if s.opts.EnableParity {
		s.navigationContext.Apply(s.active, e, phase)
	}
	if s.opts.EnableParity {
		s.views.parity.ApplyEvent(s.active, e)
		s.views.activity.ApplyEvent(s.active, e)
		if e.Type == "LoadGame" {
			s.views.parity.RestoreStartup(s.active, e, batch)
			s.views.activity.RestoreStartup(s.active, e, batch)
		}
	}
	s.views.navigation.ApplyEventPhase(s.active, e, phase)
	s.views.exploration.ApplyEvent(s.active, e)
	s.views.dirty = true
}
func (s *Service) sidecarProjectionLocked(c sidecar.Change) {
	if c.Kind == "Status" {
		s.bioStatus = nil
		if c.Valid && !s.summary.ImportInProgress && s.summary.SourceState == "live" && !s.stopped {
			s.bioStatus = append([]byte(nil), c.Raw...)
			s.captureBioLocked()
		}
	}
	if s.opts.EnableParity {
		s.views.parity.ApplySidecar(s.active, c)
		if c.Kind == "Status" && !s.summary.ImportInProgress && s.summary.SourceState == "live" {
			s.navigationContext.Status(s.active, s.views.parity.Status(s.active))
		}
		// Sidecars (Status, Cargo, ...) only feed the workspace model.
		s.markDomainsLocked(DomainWorkspace)
	}
	if c.Kind == "NavRoute" {
		s.views.navigation.ApplySidecar(c)
		s.markDomainsLocked(DomainNavigation)
	}
}
func (s *Service) resetProjectionsLocked() {
	s.startup = journal.Startup{}
	s.views.ship.ResetSource()
	s.navigationContext = navcontext.New()
	s.views.parity.ResetSource()
	s.views.activity.ResetSource()
	s.views.navigation.ResetSource()
	s.views.exploration.ResetSource()
	s.views.hasLast = false
	s.views.dirty = true
}
func (s *Service) publishProjectionsLocked() {
	key := s.projectionKeyLocked()
	whole := !s.views.hasLast || key != s.views.lastKey || s.cachedImportLocked()
	if !whole && !s.views.dirty && s.views.partial == 0 {
		return
	}
	var p Projections
	build := DomainAll
	if whole {
		p = s.projectionLocked()
		s.views.deferred = 0
	} else {
		changed := s.views.partial
		if s.views.dirty {
			changed = DomainAll
		}
		// Build only what a display (or the warm set) demands; defer the rest.
		build = changed & s.views.demand
		s.views.deferred |= changed &^ build
		s.views.deferred &^= build
		countBuilds(build)
		p = s.buildDomainsLocked(build)
	}
	first := !s.views.hasLast || key != s.views.lastKey
	s.views.dirty = false
	s.views.partial = 0
	if !s.advanceRevisionsLocked(p, build, first) && p.Source.SourceState == s.views.last.Source.SourceState && p.Source.ImportInProgress == s.views.last.Source.ImportInProgress {
		s.views.lastKey = key
		return
	}
	p.Revisions = s.views.rev
	s.views.last = p
	s.views.lastKey = key
	s.views.hasLast = true
	perfstats.Add("projection.publish", 1)
	if len(s.views.subscribers) == 0 {
		return
	}
	// Each subscriber receives its own detached snapshot (mutations never
	// cross subscribers); encoding is shared downstream by revision.
	for ch := range s.views.subscribers {
		select {
		case ch <- cloneProjection(p):
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- cloneProjection(p):
			default:
			}
		}
	}
}
func (s *Service) SubscribeProjections(ctx context.Context) (<-chan Projections, func()) {
	ch := make(chan Projections, 1)
	s.mu.Lock()
	if s.closed || len(s.views.subscribers) >= 128 {
		s.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	s.views.subscribers[ch] = struct{}{}
	ch <- s.projectionLocked()
	s.mu.Unlock()
	var once sync.Once
	finished := make(chan struct{})
	stop := func() {
		once.Do(func() {
			s.mu.Lock()
			if _, ok := s.views.subscribers[ch]; ok {
				delete(s.views.subscribers, ch)
				close(ch)
			}
			s.mu.Unlock()
			close(finished)
		})
	}
	go func() {
		select {
		case <-ctx.Done():
			stop()
		case <-s.done:
			stop()
		case <-finished:
		}
	}()
	return ch, stop
}
func (s *Service) closeProjectionsLocked() {
	for ch := range s.views.subscribers {
		delete(s.views.subscribers, ch)
		close(ch)
	}
}

func scalarCopy[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func cloneProjection(p Projections) Projections {
	q := p
	if p.Workspace != nil {
		w := *p.Workspace
		w.Parity = paritystate.Clone(w.Parity)
		w.Activity = activitystate.Clone(w.Activity)
		w.Exosuit = paritystate.CloneExosuit(w.Exosuit)
		q.Workspace = &w
	}
	q.Ship = shipstate.Clone(p.Ship)
	q.Navigation.RemainingJumps = scalarCopy(p.Navigation.RemainingJumps)
	q.Navigation.ReportedRemainingJumps = scalarCopy(p.Navigation.ReportedRemainingJumps)
	q.Navigation.Hops = append([]navstate.Hop{}, p.Navigation.Hops...)
	for i := range q.Navigation.Hops {
		q.Navigation.Hops[i].DistanceFromPreviousLY = scalarCopy(p.Navigation.Hops[i].DistanceFromPreviousLY)
	}
	q.Exploration.ReportedBodyCount = scalarCopy(p.Exploration.ReportedBodyCount)
	q.Exploration.ScannerBodyCount = scalarCopy(p.Exploration.ScannerBodyCount)
	q.Exploration.FSSProgress = scalarCopy(p.Exploration.FSSProgress)
	q.Exploration.Bodies = append([]explorestate.BodyView{}, p.Exploration.Bodies...)
	for i := range q.Exploration.Bodies {
		q.Exploration.Bodies[i] = explorestate.CloneBody(p.Exploration.Bodies[i])
	}
	return q
}

func (s *Service) Ship() shipstate.View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedImportLocked() {
		return s.projectionLocked().Ship
	}
	if s.domainFreshLocked(DomainShip) {
		perfstats.Add("getter.reuse.ship", 1)
		return shipstate.Clone(s.views.last.Ship)
	}
	return s.shipDomainLocked(s.displayFIDLocked())
}

func (s *Service) Workspace() WorkspaceView {
	s.mu.Lock()
	defer s.mu.Unlock()
	var w *WorkspaceView
	switch {
	case s.cachedImportLocked():
		w = s.projectionLocked().Workspace
	case s.domainFreshLocked(DomainWorkspace):
		perfstats.Add("getter.reuse.workspace", 1)
		w = cloneProjection(Projections{Workspace: s.views.last.Workspace}).Workspace
	default:
		w = s.workspaceDomainLocked(s.displayFIDLocked())
	}
	if w == nil {
		return WorkspaceView{}
	}
	return *w
}

func (s *Service) holdProjectionLocked(p Projections) {
	s.cachedProjectionFID = s.displayFIDLocked()
	s.cachedProjection = &p
}
