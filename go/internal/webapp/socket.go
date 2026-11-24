package webapp

import (
	"context"
	"encoding/json"
	"github.com/Jakoczig/eridion/go/internal/gamestate"
	"github.com/Jakoczig/eridion/go/internal/paritystate"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

func (s *Server) loadingStatus() map[string]any {
	marker := s.prototype()
	return map[string]any{"loadingComplete": true, "loadingInProgress": false, "numberOfFiles": 0, "numberOfEventsImported": 0, "numberOfLogLines": 0, "logSizeInBytes": 0, "offline": marker["offline"], "stage": marker["stage"]}
}

func (s *Server) socket(w http.ResponseWriter, r *http.Request) {
	token := s.sessions.token(r)
	if !s.sameOrigin(r) || !s.sessions.valid(token) {
		http.Error(w, "Forbidden", 403)
		return
	}
	// Launcher management is bound to the connection the launcher window
	// opens (scope=launcher); every other owner connection is a gameplay
	// display with exactly the paired-display capability set.
	scope := scopeGameplay
	if r.URL.Query().Get("scope") == scopeLauncher {
		scope = scopeLauncher
	}
	s.serveSocket(w, r, socketPrincipal{role: "owner", scope: scope, url: s.url, valid: func() bool { return s.sessions.valid(token) }})
}

type socketPrincipal struct {
	role, scope, url, deviceID, token string
	valid                             func() bool
	remote                            *remoteManager
	epoch                             uint64
}

func (s *Server) serveSocket(w http.ResponseWriter, r *http.Request, principal socketPrincipal) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	if principal.remote != nil {
		m := principal.remote
		m.mu.Lock()
		_, valid := m.findLocked(principal.token)
		count := 0
		for _, id := range m.connections {
			if id == principal.deviceID {
				count++
			}
		}
		if !valid || m.epoch != principal.epoch || m.http == nil || len(m.connections) >= 64 || count >= 4 {
			m.mu.Unlock()
			conn.CloseNow()
			return
		}
		m.connections[conn] = principal.deviceID
		m.mu.Unlock()
		defer func() { m.mu.Lock(); delete(m.connections, conn); m.mu.Unlock() }()
	}
	s.socketMu.Lock()
	if s.closing {
		s.socketMu.Unlock()
		_ = conn.CloseNow()
		return
	}
	s.sockets[conn] = principal.scope
	s.socketMu.Unlock()
	defer func() { _ = conn.CloseNow(); s.socketMu.Lock(); delete(s.sockets, conn); s.socketMu.Unlock() }()
	conn.SetReadLimit(1 << 20)
	readCtx, cancelRead := context.WithCancel(context.Background())
	defer cancelRead()
	go func() {
		ticker := time.NewTicker(s.pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-readCtx.Done():
				return
			case <-ticker.C:
				if !principal.valid() {
					conn.CloseNow()
					return
				}
				pingCtx, cancel := context.WithTimeout(readCtx, s.pongTimeout)
				err := conn.Ping(pingCtx)
				cancel()
				if err != nil {
					conn.CloseNow()
					return
				}
			}
		}
	}()
	var writeMu sync.Mutex
	var journalRequested atomic.Bool
	var providerRequested atomic.Bool
	var contextRequested atomic.Bool
	var spatialRequested atomic.Bool
	// communityBase is the consent revision (+1; 0 = never requested) this
	// connection last observed; set when community data are served.
	var communityBase atomic.Uint64
	var navRequested, surveyRequested, navDelivered, surveyDelivered atomic.Bool
	var shipRequested, shipDelivered atomic.Bool
	var workspaceRequested, workspaceDelivered atomic.Bool
	navigationReady := func() {
		if principal.role == "owner" && (!s.g2e || (workspaceRequested.Load() && workspaceDelivered.Load())) && (!s.g2d || (shipRequested.Load() && shipDelivered.Load())) && navRequested.Load() && surveyRequested.Load() && navDelivered.Load() && surveyDelivered.Load() {
			s.navigationReadyOnce.Do(func() { close(s.navigationReady) })
		}
	}
	send := func(value any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		if !principal.valid() {
			return context.Canceled
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		recordFrame(value, len(data))
		return conn.Write(ctx, websocket.MessageText, data)
	}
	if send(map[string]any{"name": "loadingProgress", "message": s.loadingStatus()}) != nil {
		return
	}
	// Account epoch: every connection learns the epoch it was opened under.
	// A commander switch closes gameplay connections (see accountSwitched);
	// the launcher connection stays open and is told the new epoch.
	connEpoch, epochChanged := s.account.watch()
	if send(map[string]any{"name": "accountEpoch", "message": map[string]any{"epoch": connEpoch}}) != nil {
		return
	}
	go func() {
		changed := epochChanged
		for {
			select {
			case <-readCtx.Done():
				return
			case <-changed:
				var epoch uint64
				epoch, changed = s.account.watch()
				if principal.scope != scopeLauncher {
					conn.CloseNow()
					return
				}
				if send(map[string]any{"name": "accountEpoch", "message": map[string]any{"epoch": epoch}}) != nil {
					return
				}
			}
		}
	}()
	if s.game != nil && s.g2e {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			var previous any
			var previousProviders any
			var spatialRevision uint64
			for {
				select {
				case <-readCtx.Done():
					return
				case <-ticker.C:
					if spatialRequested.Load() && s.spatial != nil {
						revision := s.spatial.Revision()
						if revision != spatialRevision {
							spatialRevision = revision
							if send(map[string]any{"name": "goSpatialChanged", "message": map[string]any{"revision": revision}}) != nil {
								return
							}
						}
					}
					// Audit F8: an EDSM consent change made in the launcher
					// reaches every open community consumer at once.
					if base := communityBase.Load(); base != 0 && s.community != nil {
						revision, enabled := s.community.Revision()
						if revision+1 != base {
							communityBase.Store(revision + 1)
							if send(map[string]any{"name": "goCommunityChanged", "message": map[string]any{"enabled": enabled}}) != nil {
								return
							}
						}
					}
					if principal.role == "owner" && providerRequested.Load() && s.providers != nil {
						v := s.providers.Status()
						if !reflect.DeepEqual(previousProviders, v) {
							previousProviders = v
							if send(map[string]any{"name": "goProviderStatusChanged", "message": v}) != nil {
								return
							}
						}
					}
					if !contextRequested.Load() {
						continue
					}
					v := s.contextView(principal)
					if !reflect.DeepEqual(previous, v) {
						previous = v
						if send(map[string]any{"name": "goNavigationContextChanged", "message": v}) != nil {
							return
						}
					}
				}
			}
		}()
	}

	if s.game != nil {
		changes, unsubscribe := s.game.Subscribe(readCtx)
		defer unsubscribe()
		go func() {
			for {
				select {
				case <-readCtx.Done():
					return
				case summary, ok := <-changes:
					if !ok {
						return
					}
					if send(map[string]any{"name": "journalSummaryChanged", "message": summary}) != nil {
						conn.CloseNow()
						return
					}
					if principal.role == "owner" && journalRequested.Load() && summary.CurrentSystem != "" {
						s.journalReadyOnce.Do(func() { close(s.journalReady) })
					}
				}
			}
		}()
	}
	if s.g2b && s.game != nil {
		changes, unsubscribe := s.game.SubscribeProjections(readCtx)
		var previous *gamestate.Projections
		defer unsubscribe()
		go func() {
			for {
				select {
				case <-readCtx.Done():
					return
				case p, ok := <-changes:
					if !ok {
						return
					}
					if (!s.g2e || previous == nil || !reflect.DeepEqual(previous.Navigation, p.Navigation)) && send(map[string]any{"name": "goNavigationChanged", "message": p.Navigation}) != nil {
						conn.CloseNow()
						return
					}
					if len(p.Navigation.Hops) > 0 && navRequested.Load() {
						navDelivered.Store(true)
					}
					if (!s.g2e || previous == nil || !reflect.DeepEqual(previous.Exploration, p.Exploration)) && send(map[string]any{"name": "goExplorationChanged", "message": p.Exploration}) != nil {
						conn.CloseNow()
						return
					}
					if len(p.Exploration.Bodies) > 0 && surveyRequested.Load() {
						surveyDelivered.Store(true)
					}
					if s.g2d && shipRequested.Load() && (!s.g2e || previous == nil || !reflect.DeepEqual(previous.Ship, p.Ship)) {
						if send(map[string]any{"name": "goShipChanged", "message": p.Ship}) != nil {
							conn.CloseNow()
							return
						}
						if p.Ship.ShipType != "" {
							shipDelivered.Store(true)
						}
					}
					if s.g2e && workspaceRequested.Load() && p.Workspace != nil {
						var old *gamestate.WorkspaceView
						if previous != nil {
							old = previous.Workspace
						}
						if !sameStaticWorkspace(old, p.Workspace) {
							if send(map[string]any{"name": "goWorkspaceChanged", "message": p.Workspace}) != nil {
								conn.CloseNow()
								return
							}
						} else if !reflect.DeepEqual(old.Parity.Status, p.Workspace.Parity.Status) {
							if send(map[string]any{"name": "goFlightChanged", "message": p.Workspace.Parity.Status}) != nil {
								conn.CloseNow()
								return
							}
						}
						if len(p.Workspace.Parity.Modules) > 0 {
							workspaceDelivered.Store(true)
						}
					}
					previous = &p
					navigationReady()

				}
			}
		}()
	}

	for {
		kind, data, err := conn.Read(readCtx)
		if err != nil {
			return
		}
		if !principal.valid() {
			_ = conn.Close(websocket.StatusPolicyViolation, "Session expired")
			return
		}
		var request struct {
			RequestID string          `json:"requestId"`
			Name      string          `json:"name"`
			Message   json.RawMessage `json:"message"`
			Epoch     *uint64         `json:"epoch"`
		}
		if kind != websocket.MessageText || json.Unmarshal(data, &request) != nil || len(request.RequestID) > 128 || len(request.Name) > 128 {
			if send(map[string]any{"error": "Invalid request"}) != nil {
				return
			}
			continue
		}
		response := map[string]any{"requestId": request.RequestID, "name": request.Name}
		// Reject anything issued under a previous commander: a gameplay
		// connection opened before the switch is no longer valid at all, and
		// a request stamped with an older epoch never runs.
		epoch := s.account.current()
		response["epoch"] = epoch
		if (principal.scope != scopeLauncher && connEpoch != epoch) || (request.Epoch != nil && *request.Epoch != epoch && !epochExempt[request.Name]) {
			response["error"] = staleEpochError
			if send(response) != nil || principal.scope != scopeLauncher {
				return
			}
			continue
		}
		navigationNonempty, surveyNonempty, shipNonempty, workspaceNonempty := false, false, false, false
		// D1: one capability set for every trusted gameplay display; launcher
		// management stays on the local launcher (see rpc_authority.go).
		if !rpcAllowed(principal, request.Name) {
			response["error"] = launcherOnlyError
			if send(response) != nil {
				return
			}
			continue
		}
		switch request.Name {
		case "getLoadingStatus":
			response["message"] = s.loadingStatus()
		case "hostInfo":
			response["message"] = map[string]any{"urls": []string{principal.url}}
		case "getGoAccountGate":
			// Gameplay displays only learn whether a launcher commander choice
			// is pending; cards, portraits and settings stay on the launcher.
			response["message"] = map[string]any{"choiceRequired": s.accountChoiceRequired()}
		case "getGoShipName":
			if s.game == nil {
				response["error"] = "Ship identity unavailable"
			} else {
				response["message"] = s.game.OwnerShipIdentity()
			}
		case "getGoGalaxyLayers":
			if s.providers == nil {
				response["message"] = map[string]any{"points": []any{}}
			} else {
				response["message"] = s.providers.Layers()
			}
		case "refreshGoCommunityGoals":
			if s.providers == nil || s.game == nil {
				response["error"] = "Consented Frontier connection required"
			} else if _, err := s.providers.Read(readCtx, "frontier", "communitygoals", ""); err != nil {
				response["error"] = err.Error()
			} else if value, err := s.operationsRPC(readCtx, "getGoOperations", json.RawMessage(`{"kind":"goals","limit":30}`)); err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "getGoOperations", "exportGoOperations", "backfillGoOperationsSource":
			if value, err := s.operationsRPC(readCtx, request.Name, request.Message); err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "getGoFieldView":
			response["message"] = s.fieldView(true)
		case "getGoBioScan":
			var req struct {
				Body string `json:"body"`
			}
			if s.game == nil || strictDecode(request.Message, &req) != nil || len(req.Body) > 20 {
				response["error"] = "Invalid BioScan body"
			} else if v, e := s.game.FieldBio(req.Body, true); e != nil {
				response["error"] = "BioScan unavailable"
			} else {
				response["message"] = v
			}
		case "getGoBuiltinModules", "setGoBuiltinModule", "importGoItinerary", "deleteGoItinerary", "getGoItinerary", "editGoItinerary", "copyGoItineraryNext", "exportGoItinerary", "getGoBGSTally", "exportGoBGSTally":
			if v, e := s.fieldRPC(readCtx, request.Name, request.Message); e != nil {
				response["error"] = e.Error()
			} else {
				response["message"] = v
			}
		case "getGoConstruction", "getGoSaleLinks", "confirmGoOrganicSale", "getGoNearest", "getGoSourceFacts", "getGoDataStatus", "queryGoHistory", "backupGoHistory", "exportGoHistory", "stageGoRestore", "rebuildGoHistory", "reimportGoSource", "getGoPublicationPolicy", "setGoPublicationPolicy", "setGoPublicationException":
			value, err := s.historyRPC(readCtx, request.Name, request.Message)
			if err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "previewGoHistorical", "confirmGoHistorical", "getGoProviders", "configureGoProvider", "setGoProviderCredential", "disconnectGoProvider", "readGoProvider", "connectGoFrontier", "completeGoFrontierReturn", "setGoFrontierClient":
			if request.Name == "getGoProviders" && principal.role == "owner" {
				providerRequested.Store(true)
			}
			if s.providers == nil {
				response["error"] = "Providers unavailable"
				break
			}
			value, err := s.providerRPC(readCtx, request.Name, request.Message)
			if err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "getGoNavigationContext":
			if !s.g2e || s.game == nil {
				response["error"] = "Navigation context unavailable"
			} else {
				contextRequested.Store(true)
				response["message"] = s.contextView(principal)
			}
		case "getGoSystem", "lookupGoSystem":
			if !s.g2e || s.game == nil || s.community == nil {
				response["error"] = "System lookup unavailable"
				break
			}
			var q struct {
				Name string `json:"name"`
			}
			if strictDecode(request.Message, &q) != nil || !cleanSystemName(q.Name) {
				response["error"] = "Invalid system name"
				break
			}
			response["message"] = s.systemView(readCtx, q.Name, request.Name == "lookupGoSystem")
		case "getGoSpatialView", "getGoSpatialStatus", "configureGoSpatial", "installGoSpatial", "installGoBubble", "removeGoBubble", "rollbackGoSpatial", "retryGoSpatial":
			if request.Name == "getGoSpatialView" {
				spatialRequested.Store(true)
			}
			if value, err := s.spatialRPC(readCtx, request.Name, request.Message, true); err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "getGoAtlasView", "getGoAtlasStatus", "configureGoAtlas", "checkGoAtlas", "installGoAtlas", "rollbackGoAtlas", "clearGoAtlasDetails", "clearGoAtlas":
			if value, err := s.atlasRPC(readCtx, request.Name, request.Message, true); err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "getGoAccounts", "setGoAccounts", "getGoCommanderOverview", "setGoAvatar", "chooseGoAvatar":
			value, err := s.accountRPC(readCtx, request.Name, request.Message)
			if err != nil {
				response["error"] = err.Error()
			} else {
				response["message"] = value
			}
		case "getGoLauncher", "setGoCompanion", "desktopAction":
			if s.companion == nil {
				response["error"] = "Native launcher unavailable"
				break
			}
			if request.Name == "setGoCompanion" {
				if err := s.saveDisplayConfig(request.Message); err != nil {
					response["error"] = err.Error()
					break
				}
			}
			if request.Name == "desktopAction" {
				var q struct {
					Action string `json:"action"`
				}
				if strictDecode(request.Message, &q) != nil {
					response["error"] = "Invalid desktop action"
					break
				}
				if q.Action != "dashboard" && q.Action != "launcher" && q.Action != "hide" && q.Action != "closeDashboard" && q.Action != "closeLauncher" && q.Action != "quit" {
					response["error"] = "Unsupported desktop action"
					break
				}
				if s.companion.action == nil {
					response["error"] = "Native desktop unavailable"
					break
				}
				if q.Action == "dashboard" && s.accountChoiceRequired() {
					response["error"] = "Choose a commander in the launcher before opening the terminal"
					break
				}
				if err := s.companion.action(q.Action); err != nil {
					response["error"] = err.Error()
					break
				}
			}
			response["message"] = s.launcherView()
			if request.Name == "getGoLauncher" {
				s.launcherOnce.Do(func() { close(s.launcherReady) })
			}
		case "getGoCommunity", "refreshGoCommunity", "setGoCommunityEnabled":
			if s.community != nil {
				revision, _ := s.community.Revision()
				communityBase.Store(revision + 1)
			}
			if s.community == nil || s.game == nil {
				response["error"] = "Community enrichment unavailable"
				break
			}
			if request.Name == "setGoCommunityEnabled" {
				var value struct {
					Enabled *bool `json:"enabled"`
				}
				decoder := json.NewDecoder(strings.NewReader(string(request.Message)))
				decoder.DisallowUnknownFields()
				if decoder.Decode(&value) != nil || value.Enabled == nil {
					response["error"] = "Expected explicit enabled setting"
					break
				}
				if err := s.community.SetEnabled(*value.Enabled); err != nil {
					response["error"] = err.Error()
					break
				}
			}
			name, address := s.game.SystemIdentity()
			if request.Name == "refreshGoCommunity" {
				response["message"] = s.community.Refresh(readCtx, name, address)
			} else {
				response["message"] = s.community.View(name, address)
			}
		case "getGoProviderGameplay":
			if s.providers == nil {
				response["message"] = []any{}
			} else {
				response["message"] = providerGameplay(s.providers.Status())
			}
		case "getGoEngineeredStatistics":
			if !s.g2e || s.game == nil {
				response["error"] = "Native statistics unavailable"
			} else {
				response["message"] = s.game.EngineeredStatistics()
			}
		case "getGoWorkspace":
			if !s.g2e || s.game == nil {
				response["error"] = "Workspace not enabled"
			} else {
				workspaceRequested.Store(true)
				value := s.game.Workspace()
				workspaceNonempty = len(value.Parity.Modules) > 0
				response["message"] = value
			}
		case "getGoShip":
			if !s.g2d || s.game == nil {
				response["error"] = "Ship journal view not enabled"
			} else {
				shipRequested.Store(true)
				value := s.game.Ship()
				shipNonempty = value.ShipType != ""
				response["message"] = value
			}
		case "getGoNavigation":
			if !s.g2b || s.game == nil {
				response["error"] = "Navigation not enabled"
			} else {
				navRequested.Store(true)
				value := s.game.Navigation()
				navigationNonempty = len(value.Hops) > 0
				response["message"] = value
			}
		case "getGoExploration":
			if !s.g2b || s.game == nil {
				response["error"] = "Exploration not enabled"
			} else {
				surveyRequested.Store(true)
				value := s.game.Exploration()
				surveyNonempty = len(value.Bodies) > 0
				response["message"] = value
			}
		case "getJournalSummary":
			if s.game == nil {
				response["error"] = "Journal ingestion disabled"
			} else {
				journalRequested.Store(true)
				response["message"] = s.game.Snapshot()
			}
		default:
			response["error"] = "Not implemented in this offline preview"
		}
		if now := s.account.current(); now != epoch {
			if epochExempt[request.Name] {
				// The account chooser itself switched the commander: report
				// the epoch its result belongs to.
				response["epoch"] = now
			} else {
				// The commander changed while this request ran: never deliver
				// a reply computed for (or across) the previous commander.
				delete(response, "message")
				response["error"] = staleEpochError
			}
		}
		if send(response) != nil {
			return
		}
		if navigationNonempty {
			navDelivered.Store(true)
		}
		if surveyNonempty {
			surveyDelivered.Store(true)
		}
		if shipNonempty {
			shipDelivered.Store(true)
		}
		if workspaceNonempty {
			workspaceDelivered.Store(true)
		}
		navigationReady()
		if request.Name == "getLoadingStatus" && principal.role == "owner" {
			s.readyOnce.Do(func() { close(s.clientReady) })
		}
		if request.Name == "getJournalSummary" && principal.role == "owner" && s.game != nil && s.game.Snapshot().CurrentSystem != "" {
			s.journalReadyOnce.Do(func() { close(s.journalReady) })
		}
	}
}

func (s *Server) handler(assets http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		if !s.trustedPeer(r) {
			http.Error(w, "Forbidden", 403)
			return
		}
		if r.URL.Path == "/auth/session" {
			s.session(w, r)
			return
		}
		if r.URL.Path == "/owner/frontier-callback" {
			s.frontierCallback(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/owner/") {
			if r.URL.Path == "/owner/journal" {
				s.ownerJournal(w, r)
			} else {
				s.ownerRemote(w, r)
			}
			return
		}
		if r.Header.Get("Upgrade") != "" {
			if r.URL.Path != "/" {
				http.Error(w, "Forbidden", 403)
				return
			}
			s.socket(w, r)
			return
		}
		if len(r.URL.Path) >= 6 && r.URL.Path[:6] == "/auth/" {
			http.Error(w, "Forbidden", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/atlas/assets/") {
			s.atlasAsset(w, r, s.sessions.valid(s.sessions.token(r)))
			return
		}
		assets.ServeHTTP(w, r)
	})
}

// Status ticks must not resend static loadouts, inventories or system bodies.
func sameStaticWorkspace(a, b *gamestate.WorkspaceView) bool {
	if a == nil || b == nil {
		return false
	}
	x, y := *a, *b
	x.Parity.Status = paritystate.StatusView{}
	y.Parity.Status = paritystate.StatusView{}
	return reflect.DeepEqual(x, y)
}
