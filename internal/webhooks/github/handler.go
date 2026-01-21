package github

import (
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v81/github"
)

// Handler validates GitHub webhooks.
type Handler struct {
	secret []byte
}

// NewHandler constructs a GitHub webhook handler.
func NewHandler(secret []byte) *Handler {
	return &Handler{secret: secret}
}

// Handle validates an incoming GitHub webhook.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	if len(h.secret) == 0 {
		http.Error(w, "webhook secret not configured", http.StatusInternalServerError)
		return nil
	}

	payload, err := gh.ValidatePayload(r, h.secret)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return err
	}

	event := gh.WebHookType(r)
	if event == "" {
		http.Error(w, "missing event header", http.StatusBadRequest)
		return fmt.Errorf("missing event header")
	}

	if _, err := gh.ParseWebHook(event, payload); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}
