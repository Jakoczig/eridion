package gamestate

import (
	"github.com/Jakoczig/eridion/go/internal/explorestate"
	"github.com/Jakoczig/eridion/go/internal/navcontext"
	"strings"
	"time"
)

func (s *Service) NavigationContext() navcontext.View {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.navigationContext.View(s.displayFIDLocked(), time.Now())
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
