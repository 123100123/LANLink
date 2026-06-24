package dashboard

import (
	"net/http"
	"time"

	"github.com/123100123/lanlink/internal/agentserver"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/discovery"
)

// handleDiscoveryScan scans the LAN for other receivers (usable as send
// targets). It is reached only through subHandler, which already restricts
// /ui/* to loopback, so the scan is never exposed to LAN clients. The agent's
// own beacon is filtered out of the results.
func handleDiscoveryScan(w http.ResponseWriter, r *http.Request) {
	cfg := config.Load()

	hosts, err := discovery.Scan(2500*time.Millisecond, cfg.DiscoveryPort)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": "error", "error": err.Error()})
		return
	}

	self := agentserver.GetState().Address
	filtered := make([]discovery.Announcement, 0, len(hosts))
	for _, h := range hosts {
		if h.Addr == self {
			continue
		}
		filtered = append(filtered, h)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "hosts": filtered})
}
