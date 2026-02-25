package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// shutdownTimeout is the maximum time to wait for the HTTP server to shut down.
const shutdownTimeout = 5 * time.Second

// Server wraps an HTTP server that receives webhook callbacks.
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	mu         sync.RWMutex
	ready      chan struct{}
	started    atomic.Bool
	logger     *slog.Logger
}

// NewServer creates a webhook server listening on the given port.
func NewServer(port int, handler *WebhookHandler, logger *slog.Logger) *Server {
	if handler == nil {
		panic("notification.NewServer: handler must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle("/webhooks/radarr", handler)
	mux.HandleFunc("/health", healthHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           mux,
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      15 * time.Second,
		},
		ready:  make(chan struct{}),
		logger: logger,
	}
}

// Ready returns a channel that is closed once the server is listening.
func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

// Addr returns the listener address once the server has started.
// Returns empty string if the server hasn't started yet.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// Start begins listening for webhook requests. It blocks until the server
// stops or an error occurs. The server shuts down gracefully when ctx is canceled.
func (s *Server) Start(ctx context.Context) error {
	if !s.started.CompareAndSwap(false, true) {
		return fmt.Errorf("webhook server already started")
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.httpServer.Addr)
	if err != nil {
		s.started.Store(false)
		return fmt.Errorf("webhook server listen: %w", err)
	}
	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()
	close(s.ready)

	s.logger.Info("webhook server started", slog.String("addr", ln.Addr().String()))

	serveDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-serveDone:
			return
		}
		s.logger.Info("webhook server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		//nolint:contextcheck // parent ctx is canceled; we need a fresh context for graceful shutdown
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("webhook server shutdown error", slog.String("error", err.Error()))
		}
	}()

	err = s.httpServer.Serve(ln)
	close(serveDone)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("webhook server: %w", err)
	}
	return nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
