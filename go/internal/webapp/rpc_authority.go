package webapp

import (
	"strings"

	"github.com/Jakoczig/eridion/go/internal/providers"
)

// Confirmed D1 (Task 13 / F16): every successfully paired display has the same
// trusted-display gameplay capabilities as the main display, for the account
// the host has authorized. Pairing, revocation, session validity and the
// host account boundary are enforced before any RPC is dispatched.
//
// Launcher management (settings, providers/credentials/consent, history
// maintenance, data installation, pairing, module enables and native window
// actions) is reachable only from the local launcher principal. Gameplay
// screens, main or paired, show no settings.
const launcherOnlyError = "Launcher only: change this in the ERIDION launcher"

var gameplayRPC = setOf(
	"getLoadingStatus", "hostInfo", "getJournalSummary", "getGoAccountGate",
	"getGoWorkspace", "getGoShip", "getGoShipName", "getGoNavigation", "getGoExploration", "getGoEngineeredStatistics",
	"getGoNavigationContext", "getGoSystem", "lookupGoSystem", "getGoGalaxyLayers", "getGoSpatialView", "getGoAtlasView",
	"getGoCommunity", "refreshGoCommunity", "refreshGoCommunityGoals", "getGoProviderGameplay",
	"getGoCommanderOverview", "getGoOperations", "exportGoOperations",
	"getGoFieldView", "getGoBioScan", "getGoConstruction", "getGoSaleLinks", "confirmGoOrganicSale", "getGoNearest",
	"getGoItinerary", "importGoItinerary", "editGoItinerary", "deleteGoItinerary", "copyGoItineraryNext", "exportGoItinerary",
	"getGoBGSTally", "exportGoBGSTally",
)

var launcherRPC = setOf(
	"getGoLauncher", "setGoCompanion", "desktopAction",
	"getGoAccounts", "setGoAccounts", "setGoAvatar", "chooseGoAvatar",
	"getGoProviders", "configureGoProvider", "setGoProviderCredential", "disconnectGoProvider", "readGoProvider", "connectGoFrontier", "completeGoFrontierReturn", "setGoFrontierClient", "previewGoHistorical", "confirmGoHistorical",
	"getGoBuiltinModules", "setGoBuiltinModule", "setGoCommunityEnabled",
	"backfillGoOperationsSource", "getGoSourceFacts", "getGoDataStatus", "queryGoHistory", "backupGoHistory", "exportGoHistory", "stageGoRestore", "rebuildGoHistory", "reimportGoSource", "getGoPublicationPolicy", "setGoPublicationPolicy", "setGoPublicationException",
	"getGoSpatialStatus", "configureGoSpatial", "installGoSpatial", "installGoBubble", "removeGoBubble", "rollbackGoSpatial", "retryGoSpatial",
	"getGoAtlasStatus", "configureGoAtlas", "checkGoAtlas", "installGoAtlas", "rollbackGoAtlas", "clearGoAtlasDetails", "clearGoAtlas",
)

func setOf(names ...string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

// rpcAllowed reports whether principal may call name. Unknown names fall
// through to the dispatcher, which rejects them.
func rpcAllowed(principal socketPrincipal, name string) bool {
	if launcherRPC[name] {
		return principal.role == "owner" && principal.scope == scopeLauncher
	}
	return true
}

// providerGameplay is the bounded provider DTO shared by every gameplay
// display: provider identity, coarse state and the cached snapshot of the
// selected commander. Configuration, credentials, queues and error text stay
// on the launcher surface.
func providerGameplay(all []providers.Status) []map[string]any {
	out := []map[string]any{}
	for _, p := range all {
		// readReady is a derived boolean; configuration and credentials stay
		// on the launcher.
		row := map[string]any{"id": p.Capability.ID, "state": p.State, "lastSuccess": p.LastSuccess, "readReady": p.Config.Read && !p.Config.Paused && p.Capability.Gate == "" && p.CredentialStored}
		// Only the selected commander's snapshot, or public goal data.
		if p.Snapshot != nil && (p.Snapshot.ActiveCommander || strings.Contains(p.Snapshot.Source, "communitygoals")) {
			row["snapshot"] = map[string]any{"activeCommander": p.Snapshot.ActiveCommander, "source": p.Snapshot.Source, "fetchedAt": p.Snapshot.FetchedAt, "expiresAt": p.Snapshot.ExpiresAt, "facts": p.Snapshot.Facts}
		}
		out = append(out, row)
	}
	return out
}
