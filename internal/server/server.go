package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"helixops/internal/config"
	"helixops/internal/orchestrator"
	"helixops/internal/analyzer"
	"helixops/internal/clients/prometheus"
	"helixops/internal/clients/github"
	"helixops/internal/clients/loki"
	"helixops/pkg/llm"
)

// Server wraps the HTTP server and dependencies
type Server struct {
	cfg     *config.Config
	srv     *http.Server
	handler *Handler
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Initialize clients
	promClient := prometheus.NewClient(cfg.Prometheus.URL, cfg.Prometheus.GetTimeoutDuration())
	githubClient := github.NewClient(cfg.GitHub.APIURL, cfg.GitHub.Token)
	lokiClient := loki.NewClient(cfg.Loki.URL, cfg.Loki.GetTimeoutDuration())

	// Initialize LLM provider
	llmProvider, err := llm.NewProvider(cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	// Initialize orchestrator
	orch := orchestrator.New(promClient, githubClient, lokiClient, cfg)

	// Initialize analyzer
	anlz := analyzer.New(llmProvider)

	// Create handler
	handler := NewHandler(cfg, orch, anlz)

	// Create router
	router := SetupRouter(handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		cfg:     cfg,
		srv:     srv,
		handler: handler,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Server listening on %s", s.srv.Addr)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	os.Exit(0)
}
