// Package server sets up the HTTP router, endpoints, and core webhook serving logic.
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
	"log/slog"

	"helixops/internal/config"
	"helixops/internal/orchestrator"
	"helixops/internal/analyzer"
	"helixops/internal/clients/prometheus"
	"helixops/internal/clients/github"
	"helixops/internal/clients/loki"
	"helixops/internal/clients/tempo"
	"helixops/internal/output"
	"helixops/internal/postmortem"
	"helixops/internal/remediation"
	"helixops/pkg/llm"
)

// Server encapsulates the HTTP server instance and its registered dependencies.
type Server struct {
	cfg     *config.Config
	srv     *http.Server
	handler *Handler
}

// New initializes a complete Server instance, bootstrapping all clients and handlers.
func New(cfg *config.Config) *Server {
	// Initialize clients
	promClient := prometheus.NewClient(cfg.Prometheus.URL, cfg.Prometheus.GetTimeoutDuration())
	githubClient := github.NewClient(cfg.GitHub.APIURL, cfg.GitHub.Token)
	lokiClient := loki.NewClient(cfg.Loki.URL, cfg.Loki.GetTimeoutDuration())
	
	// Optional Tempo client
	var tempoClient *tempo.Client
	if cfg.Tempo.Enabled {
		logger := slog.Default() // basic logger
		tempoClient = tempo.NewClient(cfg.Tempo.URL, cfg.Prometheus.GetTimeoutDuration(), logger)
	}

	// Initialize LLM provider
	llmProvider, err := llm.NewProvider(cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	// Initialize orchestrator
	orch := orchestrator.New(promClient, githubClient, lokiClient, tempoClient, cfg)

	// Initialize analyzer
	anlz := analyzer.New(llmProvider)

	// Initialize Remediation Engine and Postmortem Generator
	importRemediation := "helixops/internal/remediation"
	_ = importRemediation // pseudo bypass compiler check until imports added
	rulesEngine := remediation.NewEngine()
	generator := postmortem.NewGenerator(llmProvider, rulesEngine)
	mdReporter := output.NewMarkdownReporterFromConfig(cfg.Output.Markdown)

	// Create handler
	handler := NewHandler(cfg, orch, anlz, generator, mdReporter)

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

// Start begins listening for incoming HTTP requests in a blocking manner on the configured port.
func (s *Server) Start() error {
	log.Printf("Server listening on %s", s.srv.Addr)
	return s.srv.ListenAndServe()
}

// Shutdown initiates a graceful termination of the HTTP server, ensuring all active connections finish before exiting.
func (s *Server) Shutdown() {
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	os.Exit(0)
}
