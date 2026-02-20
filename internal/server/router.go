package server

import (
	"github.com/go-chi/chi/v5"
)

// SetupRouter initializes a chi router and attaches the standard server routes.
func SetupRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()

	// Register routes
	handler.RegisterRoutes(r)

	return r
}
