package server

import (
	"github.com/go-chi/chi/v5"
)

// SetupRouter creates and configures the HTTP router
func SetupRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()

	// Register routes
	handler.RegisterRoutes(r)

	return r
}
