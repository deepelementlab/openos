package server

import (
	"net/http"
	"github.com/agentos/aos/internal/config"
)

// GetHandler returns the HTTP handler for testing
func (s *Server) GetHandler() http.Handler {
	return s.server.Handler
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *config.Config {
	return s.config
}
