package dashboard

import (
	"net/http"
	"os"
	"sync"
	"time"
)

// The dashboard (web) build shuts itself down a few seconds after the browser
// tab is closed, freeing the port. The page sends a beacon to /ui/shutdown on
// pagehide; that arms a short timer. The normal /ui/state poll cancels the timer
// — so a refresh/navigation (which re-polls within the grace window) does NOT
// kill the agent, but actually closing the tab (polling stops) does.

const shutdownGrace = 3 * time.Second

var (
	shutdownMu    sync.Mutex
	shutdownTimer *time.Timer
)

func armShutdown() {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()
	if shutdownTimer != nil {
		shutdownTimer.Stop()
	}
	shutdownTimer = time.AfterFunc(shutdownGrace, func() {
		os.Exit(0)
	})
}

func cancelShutdown() {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()
	if shutdownTimer != nil {
		shutdownTimer.Stop()
		shutdownTimer = nil
	}
}

func handleShutdown(w http.ResponseWriter, r *http.Request) {
	armShutdown()
	w.WriteHeader(http.StatusNoContent)
}
