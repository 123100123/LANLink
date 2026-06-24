package main

import (
	"log"
	"time"

	"github.com/123100123/lanlink/agent/dashboard"
	"github.com/123100123/lanlink/internal/agentserver"
)

// The agent binary is the dashboard build of LANLink: it runs the pure-Go
// receiver core (internal/agentserver) and layers the web dashboard
// (agent/dashboard + agent-web) on top. The pure terminal binary cmd/lanlink
// runs the same core without importing any UI package.
func main() {
	err := agentserver.Run(agentserver.Options{
		RegisterRoutes: dashboard.RegisterRoutes,
		OnListening: func(port string) {
			log.Println("Dashboard: http://127.0.0.1:" + port + "/ui")
			log.Println("")
			if dashboard.ShouldOpenDashboard() {
				go func() {
					time.Sleep(300 * time.Millisecond)
					dashboard.OpenDashboard(port)
				}()
			}
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
