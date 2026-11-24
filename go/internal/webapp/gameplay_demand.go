package webapp

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"github.com/Jakoczig/eridion/go/internal/gamestate"
	"github.com/coder/websocket"
)

// Gameplay demand (G3K PERF-06). Each connection may declare, with
// setGoDemand, which live topics its visible pages need. A replace carries a
// strictly increasing generation; visible=false (hidden or minimized) or an
// empty topic list releases them. The host prepares the union of demand from
// all connections plus, when the launcher's low-latency mode is on, a bounded
// warm set. A connection that never declares demand keeps the legacy
// behaviour (every projection), so older clients are unaffected.
//
// Keyboard focus is not part of demand: a visible paired screen stays live
// while the game has focus.

type topic uint16

const (
	topicNavigation topic = 1 << iota
	topicExploration
	topicShip
	topicWorkspace
	topicContext
	topicProviders
	topicSpatial
	topicCommunity
	viewTopics = topicNavigation | topicExploration | topicShip | topicWorkspace
)

var topicNames = map[string]topic{"navigation": topicNavigation, "exploration": topicExploration, "ship": topicShip, "workspace": topicWorkspace, "context": topicContext, "providers": topicProviders, "spatial": topicSpatial, "community": topicCommunity}

type connDemand struct {
	declared, visible bool
	topics            topic
	generation        uint64
}

// allows reports whether the connection currently wants t pushed.
func (d connDemand) allows(t topic) bool { return d.declared && d.visible && d.topics&t != 0 }

type demandRegistry struct {
	// apply serializes compute+apply so concurrent recomputes reach the game
	// service in order (review R1). Lock order: apply, then mu, then the
	// game service; the game service never calls back into demand.
	apply sync.Mutex
	mu    sync.Mutex
	conns map[*websocket.Conn]connDemand
	last  uint8
	set   bool
}

// parseDemand validates a setGoDemand request.
func parseDemand(raw json.RawMessage) (connDemand, error) {
	var req struct {
		Generation json.Number `json:"generation"`
		Visible    bool        `json:"visible"`
		Topics     []string    `json:"topics"`
	}
	if strictDecode(raw, &req) != nil || len(req.Topics) > 16 {
		return connDemand{}, errors.New("Invalid demand")
	}
	g, err := strconv.ParseUint(string(req.Generation), 10, 64)
	if err != nil || g == 0 {
		return connDemand{}, errors.New("Invalid demand generation")
	}
	d := connDemand{declared: true, visible: req.Visible, generation: g}
	for _, name := range req.Topics {
		t, ok := topicNames[name]
		if !ok {
			return connDemand{}, errors.New("Unknown demand topic")
		}
		d.topics |= t
	}
	return d, nil
}

// updateDemand applies a connection's replace (or removes it when release
// is true) and recomputes the host demand. A replace with a generation not
// newer than the current one is ignored (late or duplicate).
func (s *Server) updateDemand(conn *websocket.Conn, d connDemand, remove bool) (connDemand, bool) {
	s.demand.mu.Lock()
	if s.demand.conns == nil {
		s.demand.conns = map[*websocket.Conn]connDemand{}
	}
	cur := s.demand.conns[conn]
	applied := false
	if remove {
		delete(s.demand.conns, conn)
	} else if !cur.declared || d.generation > cur.generation {
		s.demand.conns[conn] = d
		cur, applied = d, true
	}
	s.demand.mu.Unlock()
	s.recomputeDemand()
	return cur, applied
}

// registerDemand adds a connection with legacy (undeclared) demand.
func (s *Server) registerDemand(conn *websocket.Conn) {
	s.demand.mu.Lock()
	if s.demand.conns == nil {
		s.demand.conns = map[*websocket.Conn]connDemand{}
	}
	s.demand.conns[conn] = connDemand{}
	s.demand.mu.Unlock()
	s.recomputeDemand()
}

func (s *Server) connDemand(conn *websocket.Conn) connDemand {
	s.demand.mu.Lock()
	defer s.demand.mu.Unlock()
	return s.demand.conns[conn]
}

// lowLatency reports the launcher's "Low latency: keep views ready" setting.
func (s *Server) lowLatency() bool {
	if s.companion == nil {
		return false
	}
	s.companion.mu.Lock()
	defer s.companion.mu.Unlock()
	return s.companion.config.LowLatency
}

// recomputeDemand pushes the union of connection demand (plus the warm set)
// to the game service, which builds only demanded domains.
func (s *Server) recomputeDemand() {
	if s.game == nil {
		return
	}
	s.demand.apply.Lock()
	defer s.demand.apply.Unlock()
	union := topic(0)
	if s.lowLatency() {
		union |= viewTopics // bounded warm set: the four projection domains
	}
	s.demand.mu.Lock()
	for _, d := range s.demand.conns {
		switch {
		case !d.declared:
			union |= viewTopics // legacy connection: everything, as before
		case d.visible:
			union |= d.topics
		}
	}
	mask := uint8(0)
	for t, dom := range map[topic]uint8{topicNavigation: gamestate.DomainNavigation, topicExploration: gamestate.DomainExploration, topicShip: gamestate.DomainShip, topicWorkspace: gamestate.DomainWorkspace} {
		if union&t != 0 {
			mask |= dom
		}
	}
	changed := !s.demand.set || mask != s.demand.last
	s.demand.last, s.demand.set = mask, true
	s.demand.mu.Unlock()
	if changed {
		s.game.SetDemand(mask)
	}
}

// encodedPayloads shares one JSON encoding of a published domain value among
// every connection (keyed by message name and revision).
type encodedPayloads struct {
	mu      sync.Mutex
	entries map[string]encodedPayload
}

type encodedPayload struct {
	revision uint64
	frame    []byte
}

// frame returns the encoded frame for the value published under key at
// revision. Equal key and revision always denote the same value.
func (e *encodedPayloads) frame(key, name string, revision uint64, value any) ([]byte, error) {
	// Encoding under the lock makes concurrent sockets share one encoding
	// (single flight); payloads are small and the lock is never held while
	// writing to a connection.
	if revision == 0 {
		// An unpublished snapshot (a connection's initial projection) has no
		// identity to share: encode it for this connection only.
		return json.Marshal(map[string]any{"name": name, "message": value})
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if p, ok := e.entries[key]; ok && p.revision == revision {
		recordShared(name)
		return p.frame, nil
	}
	data, err := json.Marshal(map[string]any{"name": name, "message": value})
	if err != nil {
		return nil, err
	}
	if e.entries == nil || len(e.entries) > 64 {
		e.entries = map[string]encodedPayload{} // bounded: keys are a few names
	}
	e.entries[key] = encodedPayload{revision: revision, frame: data}
	return data, nil
}
