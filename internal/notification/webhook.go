package notification

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
)

// maxBodySize limits the webhook request body to 1 MB.
const maxBodySize = 1 << 20

// WebhookHandler handles incoming Radarr webhook requests.
type WebhookHandler struct {
	service *Service
	secret  string
	logger  *slog.Logger
}

// NewWebhookHandler creates an HTTP handler for Radarr webhooks.
func NewWebhookHandler(service *Service, secret string, logger *slog.Logger) *WebhookHandler {
	if service == nil {
		panic("notification.NewWebhookHandler: service must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{
		service: service,
		secret:  secret,
		logger:  logger,
	}
}

// ServeHTTP handles POST /webhooks/radarr requests.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.validateSecret(r) {
		h.logger.Warn("webhook request with invalid secret")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload RadarrWebhookPayload
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error("failed to decode webhook payload", slog.String("error", err.Error()))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.logger.Info("received radarr webhook",
		slog.String("event", payload.EventType),
		slog.String("movie", payload.MovieTitle()),
	)

	if payload.EventType != EventDownload {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.service.NotifyDownloadComplete(r.Context(), &payload); err != nil {
		h.logger.Error("notification failed", slog.String("error", err.Error()))
	}

	w.WriteHeader(http.StatusOK)
}

// validateSecret checks the webhook secret if one is configured.
func (h *WebhookHandler) validateSecret(r *http.Request) bool {
	if h.secret == "" {
		return true
	}
	provided := r.Header.Get("X-Webhook-Secret")
	return subtle.ConstantTimeCompare([]byte(h.secret), []byte(provided)) == 1
}
