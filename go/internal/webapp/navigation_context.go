package webapp

import (
	"context"
	"github.com/Jakoczig/eridion/go/internal/community"
	"github.com/Jakoczig/eridion/go/internal/navcontext"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
	"strings"
	"sync"
	"time"
)

func strictDecode(raw []byte, v any) error { return navcontext.Decode(raw, v) }

// contextView returns the shared navigation context with the instant it next
// changes by time alone (zero: only an event changes it).
func (s *Server) contextView(p socketPrincipal) (any, time.Time) {
	v, until := s.navContext.get(s.game, s.account.current())
	return map[string]any{"navigation": v, "policy": s.displayPolicy(p)}, until
}

// freshContextView builds the context now: an explicit read (for example
// right after a jump) never receives a shared copy (review R6).
func (s *Server) freshContextView(p socketPrincipal) any {
	perfstats.Add("context.build", 1)
	return map[string]any{"navigation": s.game.NavigationContext(), "policy": s.displayPolicy(p)}
}

// navContextMemo shares one navigation-context build among every connection
// (G3K PERF-06). G3L: the copy is keyed by the context signal sequence and
// stays valid until an input changes or its time deadline passes, instead of
// a 900 ms TTL; an account epoch change invalidates it.
type navContextMemo struct {
	mu    sync.Mutex
	epoch uint64
	seq   uint64
	until time.Time
	value navcontext.View
	set   bool
}

type navContextSource interface {
	NavigationContextSignal() (uint64, <-chan struct{})
	NavigationContextSnapshot() (navcontext.View, uint64, time.Time)
}

func (m *navContextMemo) get(src navContextSource, epoch uint64) (navcontext.View, time.Time) {
	seq, _ := src.NavigationContextSignal()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.set && m.epoch == epoch && m.seq == seq && (m.until.IsZero() || time.Now().Before(m.until)) {
		perfstats.Add("context.reuse", 1)
		return m.value, m.until
	}
	perfstats.Add("context.build", 1)
	m.value, m.seq, m.until = src.NavigationContextSnapshot()
	m.epoch, m.set = epoch, true
	return m.value, m.until
}
func (s *Server) systemView(ctx context.Context, name string, lookup bool) any {
	p, local := s.game.KnownSystem(name)
	var external community.View
	if lookup {
		if p.Address != "" && p.Coordinates == nil {
			// Task 4 / F05: a witnessed system without StarPos resolves its
			// coordinates through the consented lookup, bound to its address.
			external = s.community.Resolve(ctx, p.Name, p.Address)
		} else if p.Address != "" {
			external = s.community.Refresh(ctx, p.Name, p.Address)
		} else {
			external = s.community.Lookup(ctx, name)
		}
	} else {
		if p.Address != "" {
			external = s.community.View(p.Name, p.Address)
		} else {
			external = s.community.Named(name)
		}
	}
	// Fill only missing coordinates, and only for the same identity; witnessed
	// coordinates are never replaced. Passive reads use cached data only.
	if p.Name != "" && p.Coordinates == nil && external.Coordinates != nil && p.Address != "" && external.SystemAddress == p.Address {
		c := *external.Coordinates
		p.Coordinates, p.CoordinatesSource, p.CoordinatesFetchedAt = &c, "EDSM", external.FetchedAt
	}
	if p.Name == "" && external.SystemAddress != "" {
		p = navcontext.Point{Name: external.SystemName, Address: external.SystemAddress, Coordinates: external.Coordinates, Source: "EDSM"}
	}
	availability := "unavailable"
	if local.SystemName != "" && len(local.Bodies) > 0 {
		availability = "local_cached"
	} else if len(external.Bodies) > 0 {
		availability = "community_cached"
	}
	return map[string]any{"point": p, "local": local, "community": external, "availability": availability}
}
func cleanSystemName(name string) bool {
	return name != "" && len(name) <= 128 && !strings.ContainsAny(name, "\r\n\x00")
}
func (s *Server) LauncherReady() <-chan struct{} { return s.launcherReady }
