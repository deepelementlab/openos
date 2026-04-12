package routes

import (
	"net/http"
	"strings"

	agenthandler "github.com/agentos/aos/api/handlers/agent"
	monitoringhandler "github.com/agentos/aos/api/handlers/monitoring"
)

// Handlers groups all HTTP handlers for centralised route registration.
type Handlers struct {
	Agent      *agenthandler.AgentHandler
	Monitoring *monitoringhandler.MonitoringHandler
}

// RegisterRoutes registers all application routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, h Handlers) {
	// Health & monitoring
	if h.Monitoring != nil {
		mux.HandleFunc("/health", methodGet(h.Monitoring.Health))
		mux.HandleFunc("/ready", methodGet(h.Monitoring.Ready))
		mux.HandleFunc("/live", methodGet(h.Monitoring.Live))
		mux.HandleFunc("/metrics", methodGet(h.Monitoring.Metrics))
	}

	// Agent CRUD and lifecycle operations
	if h.Agent != nil {
		// List and Create agents
		mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				h.Agent.List(w, r)
			case http.MethodPost:
				h.Agent.Create(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// Agent operations by ID
		mux.HandleFunc("/api/v1/agents/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			
			// Handle action endpoints
			if strings.HasSuffix(path, "/start") {
				if r.Method == http.MethodPost {
					h.Agent.Start(w, r)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			if strings.HasSuffix(path, "/stop") {
				if r.Method == http.MethodPost {
					h.Agent.Stop(w, r)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			if strings.HasSuffix(path, "/restart") {
				if r.Method == http.MethodPost {
					h.Agent.Restart(w, r)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			
			// Handle regular CRUD operations
			switch r.Method {
			case http.MethodGet:
				h.Agent.Get(w, r)
			case http.MethodPut:
				h.Agent.Update(w, r)
			case http.MethodDelete:
				h.Agent.Delete(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}
}

// methodGet wraps a handler function to only accept GET requests.
func methodGet(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fn(w, r)
	}
}
