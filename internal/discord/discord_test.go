package discord

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendEmbed(t *testing.T) {
	var got EmbedPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := SendEmbed(srv.URL, "search", "title", "10.00", "https://example.com", "https://example.com/i.jpg"); err != nil {
		t.Fatal(err)
	}
	if len(got.Embeds) != 1 || got.Embeds[0].Title != "title" || got.Embeds[0].Image == nil {
		t.Errorf("unexpected payload: %+v", got)
	}
}

func TestSendEmbedRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	err := SendEmbed(srv.URL, "s", "t", "1", "u", "")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestSendEmbedErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unknown Webhook", http.StatusNotFound)
	}))
	defer srv.Close()

	err := SendEmbed(srv.URL, "s", "t", "1", "u", "")
	if err == nil || errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected non-rate-limit error, got %v", err)
	}
}
