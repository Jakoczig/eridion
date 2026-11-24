package gamestate

import (
	"reflect"
	"time"

	"github.com/Jakoczig/eridion/go/internal/paritystate"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
)

// Projection domains (G3K PERF-05/06). A change marks only the domains it
// can affect; a publication rebuilds only changed domains that some display
// (or the low-latency warm set) demands; the others are deferred and built
// when demanded or read.
const (
	DomainNavigation uint8 = 1 << iota
	DomainExploration
	DomainShip
	DomainWorkspace
	DomainAll = DomainNavigation | DomainExploration | DomainShip | DomainWorkspace
)

// Revisions are semantic per-domain revisions: a counter advances only when
// the published value of that domain changed. Flight is the live Status part
// of the workspace; Workspace is the rest.
type Revisions struct {
	Navigation, Exploration, Ship, Workspace, Flight uint64
}

// projectionKey is the service state every domain's presentation depends on
// (displayed commander, active commander, source state). A change in it
// rebuilds every domain.
type projectionKey struct {
	fid, active, sourceState string
	importing, cached        bool
}

func (s *Service) projectionKeyLocked() projectionKey {
	return projectionKey{fid: s.displayFIDLocked(), active: s.active, sourceState: s.summary.SourceState, importing: s.summary.ImportInProgress, cached: s.summary.Cached}
}

// cachedImportLocked: during an import the service shows the held cached
// projection, which is always built whole.
func (s *Service) cachedImportLocked() bool {
	return s.cachedProjection != nil && s.summary.ImportInProgress && s.displayFIDLocked() != "" && s.displayFIDLocked() == s.cachedProjectionFID
}

// markDomainsLocked records a change confined to the given domains.
func (s *Service) markDomainsLocked(mask uint8) { s.views.partial |= mask }

// domainFreshLocked reports whether the last published value of a domain is
// current, so a read can reuse it instead of rebuilding.
func (s *Service) domainFreshLocked(bit uint8) bool {
	if bit == DomainWorkspace {
		// Frontier module evidence expires after its cache TTL even
		// without a journal or sidecar change (review R4).
		if c, ok := s.frontierModules[s.displayFIDLocked()]; ok && time.Since(c.at) >= 30*time.Second {
			return false
		}
	}
	return s.views.hasLast && !s.views.dirty && s.views.partial&bit == 0 && s.views.deferred&bit == 0 && s.views.lastKey == s.projectionKeyLocked() && !s.cachedImportLocked()
}

// SetDemand sets the union of domains that connected displays (plus the
// low-latency warm set) need prepared. Domains deferred while undemanded are
// built at once when they become demanded.
func (s *Service) SetDemand(mask uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mask &= DomainAll
	added := mask &^ s.views.demand
	s.views.demand = mask
	if late := added & s.views.deferred; late != 0 {
		s.views.deferred &^= late
		s.views.partial |= late
		s.publishProjectionsLocked()
	}
}

// Demand returns the current demand mask (tests and diagnostics).
func (s *Service) Demand() uint8 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.views.demand
}

func staticWorkspaceEqual(a, b *WorkspaceView) bool {
	if a == nil || b == nil {
		return a == b
	}
	x, y := *a, *b
	x.Parity.Status = paritystate.StatusView{}
	y.Parity.Status = paritystate.StatusView{}
	return reflect.DeepEqual(x, y)
}

func flightOf(w *WorkspaceView) paritystate.StatusView {
	if w == nil {
		return paritystate.StatusView{}
	}
	return w.Parity.Status
}

// buildDomainsLocked returns the next projection: domains in build are
// rebuilt, the others keep their last published value.
func (s *Service) buildDomainsLocked(build uint8) Projections {
	fid := s.displayFIDLocked()
	p := s.views.last
	p.Source = s.selectedSummaryLocked()
	if build&DomainShip != 0 {
		p.Ship = s.shipDomainLocked(fid)
	}
	if build&DomainNavigation != 0 {
		p.Navigation = s.navigationDomainLocked(fid)
	}
	if build&DomainExploration != 0 {
		p.Exploration = s.explorationDomainLocked(fid)
	}
	if build&DomainWorkspace != 0 {
		p.Workspace = s.workspaceDomainLocked(fid)
	}
	return p
}

// advanceRevisionsLocked bumps the revision of every domain whose value
// changed and reports whether anything did.
func (s *Service) advanceRevisionsLocked(p Projections, build uint8, first bool) bool {
	last := s.views.last
	changed := false
	bump := func(bit uint8, rev *uint64, differs func() bool) {
		if build&bit != 0 && (first || differs()) {
			*rev++
			changed = true
		}
	}
	bump(DomainNavigation, &s.views.rev.Navigation, func() bool { return !reflect.DeepEqual(p.Navigation, last.Navigation) })
	bump(DomainExploration, &s.views.rev.Exploration, func() bool { return !reflect.DeepEqual(p.Exploration, last.Exploration) })
	bump(DomainShip, &s.views.rev.Ship, func() bool { return !reflect.DeepEqual(p.Ship, last.Ship) })
	if build&DomainWorkspace != 0 {
		if first || !staticWorkspaceEqual(p.Workspace, last.Workspace) {
			s.views.rev.Workspace++
			changed = true
		}
		if first || !reflect.DeepEqual(flightOf(p.Workspace), flightOf(last.Workspace)) {
			s.views.rev.Flight++
			changed = true
		}
	}
	return changed
}

func countBuilds(build uint8) {
	for _, d := range []struct {
		bit  uint8
		name string
	}{{DomainNavigation, "navigation"}, {DomainExploration, "exploration"}, {DomainShip, "ship"}, {DomainWorkspace, "workspace"}} {
		if build&d.bit == 0 {
			perfstats.Add("domain.deferred."+d.name, 1)
		}
	}
}
