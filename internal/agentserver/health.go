package agentserver

import (
	"encoding/json"
	"net/http"

	"github.com/123100123/lanlink/protocol"
)

func healthHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	response := protocol.HealthResponse{
		Status:  "ok",
		Service: "lanlink-agent",
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
	}
}
