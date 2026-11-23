package webapp

import (
	"sync"
	"sync/atomic"
)

// Connection scopes. The launcher window opens its socket with
// scope=launcher; every other owner connection is a gameplay display and gets
// exactly the paired-display capability set. Paired displays use viewer.
const (
	scopeLauncher = "launcher"
	scopeGameplay = "gameplay"
	scopeViewer   = "viewer"
)

const staleEpochError = "Commander changed; this view was refreshed for the selected commander. Retry the action."

// epochExempt lists RPCs that stay valid across a commander change: loading
// status, host information and the launcher's own account chooser, which
// names its target commander explicitly.
var epochExempt = setOf("getLoadingStatus", "hostInfo", "getGoAccounts", "setGoAccounts")

// accountEpochState is the host's account epoch. It advances whenever the
// commander shown on every display changes away from a known commander; each
// advance closes gameplay connections and signals the launcher connection.
type accountEpochState struct {
	mu      sync.Mutex
	fid     string
	epoch   atomic.Uint64
	changed chan struct{}
}

func (a *accountEpochState) init() {
	a.epoch.Store(1)
	a.changed = make(chan struct{})
}

func (a *accountEpochState) current() uint64 { return a.epoch.Load() }

// watch returns the current epoch and a channel closed on its next change.
func (a *accountEpochState) watch() (uint64, <-chan struct{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.epoch.Load(), a.changed
}

// observe records the displayed commander and reports whether the epoch
// advanced. The first binding (no previous commander) shows no old data and
// does not advance it.
func (a *accountEpochState) observe(fid string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	previous := a.fid
	a.fid = fid
	if previous == "" || previous == fid {
		return false
	}
	a.epoch.Add(1)
	close(a.changed)
	a.changed = make(chan struct{})
	return true
}

// accountSwitched applies a displayed-commander change: a new epoch, and every
// gameplay connection (main and paired displays) is closed so pending client
// callbacks are cancelled and the displays reconnect under the new commander.
// The launcher connection stays open; it receives the new epoch instead.
func (s *Server) accountSwitched(fid string) {
	if !s.account.observe(fid) {
		return
	}
	s.socketMu.Lock()
	defer s.socketMu.Unlock()
	for conn, scope := range s.sockets {
		if scope != scopeLauncher {
			_ = conn.CloseNow()
		}
	}
}

// launcherHTTP reports whether an owner HTTP management request came from the
// launcher window, which marks its requests with the launcher scope header.
func launcherHTTP(scope string) bool { return scope == scopeLauncher }
