package gamestate

import (
	"github.com/Jakoczig/eridion/go/internal/explorestate"
	"github.com/Jakoczig/eridion/go/internal/navcontext"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
	"strings"
	"time"
)

func (s *Service) NavigationContext() navcontext.View {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.navigationContextLocked(time.Now())
}
func (s *Service) navigationContextLocked(now time.Time) navcontext.View {
	v := s.navigationContext.View(s.displayFIDLocked(), now)
	if s.summary.ImportInProgress || s.summary.Cached || s.summary.SourceState != "live" {
		v.Fresh = false
		v.Mode = "idle"
	}
	// Reserve current/destination and remaining legs before bounded historical context.
	history := v.Points
	v.Points = nil
	seen := map[string]bool{}
	add := func(p navcontext.Point) {
		if len(v.Points) < 192 && !seen[p.Address] {
			v.Points = append(v.Points, p)
			seen[p.Address] = true
		}
	}
	if v.Current != nil {
		add(*v.Current)
	}
	positions := s.views.navigation.VerifiedPositions(s.displayFIDLocked())
	hops := s.views.navigation.View(s.displayFIDLocked()).Hops
	start := 0
	for i, h := range hops {
		if h.IsCurrent {
			start = i
		}
		if h.IsNext {
			start = i
			break
		}
	}
	addPosition := func(i int) {
		p := positions[i]
		x := p.Coordinates
		add(navcontext.Point{Name: p.Name, Address: p.Address, Coordinates: &x, Source: "verified_route"})
	}
	if len(positions) > 0 {
		addPosition(len(positions) - 1)
	}
	for i := start; i < len(positions); i++ {
		addPosition(i)
	}
	for _, p := range history {
		add(p)
	}

	return v
}
func (s *Service) KnownSystem(name string) (navcontext.Point, explorestate.ExplorationView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var p navcontext.Point
	for _, x := range s.navigationContext.View(s.displayFIDLocked(), time.Now()).Points {
		if strings.EqualFold(x.Name, name) {
			p = x
			break
		}
	}
	if p.Name == "" {
		for _, x := range s.views.navigation.VerifiedPositions(s.displayFIDLocked()) {
			if strings.EqualFold(x.Name, name) {
				coords := x.Coordinates
				p = navcontext.Point{Name: x.Name, Address: x.Address, Coordinates: &coords, Source: "verified_route"}
				break
			}
		}
	}
	if p.Name == "" {
		return p, explorestate.ExplorationView{}
	}
	return p, s.views.exploration.CachedSystem(s.displayFIDLocked(), p.Address)
}

// navContextSignal makes navigation-context delivery event-driven (G3L): it
// is advanced only when an input of NavigationContext changes (journal event,
// Status flight mode, NavRoute, source reset/restore, display commander or
// source state), checked at the single projection publication point. The
// only scheduled refresh left is the time-dependent mode window (Deadline).
type navContextSignal struct {
	gen     uint64 // bumped at every input mutation site
	key     navContextKey
	seq     uint64
	changed chan struct{}
}
type navContextKey struct {
	gen                    uint64
	fid, source            string
	importing, cached, set bool
}

func (s *Service) signalNavigationContextLocked() {
	k := navContextKey{gen: s.navSignal.gen, fid: s.displayFIDLocked(), source: s.summary.SourceState, importing: s.summary.ImportInProgress, cached: s.summary.Cached, set: true}
	if k == s.navSignal.key {
		return
	}
	s.navSignal.key = k
	s.navSignal.seq++
	if s.navSignal.changed != nil {
		close(s.navSignal.changed)
		s.navSignal.changed = nil
	}
	perfstats.Add("context.signal", 1)
}

// NavigationContextSignal returns the current context sequence and a channel
// closed when it advances. It never builds the context.
func (s *Service) NavigationContextSignal() (uint64, <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.navSignal.changed == nil {
		s.navSignal.changed = make(chan struct{})
	}
	return s.navSignal.seq, s.navSignal.changed
}

// NavigationContextSnapshot builds the context with the sequence it reflects
// and the instant it next changes by time alone (zero: never by time).
func (s *Service) NavigationContextSnapshot() (navcontext.View, uint64, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	v := s.navigationContextLocked(now)
	return v, s.navSignal.seq, s.navigationContext.Deadline(s.displayFIDLocked(), now)
}
