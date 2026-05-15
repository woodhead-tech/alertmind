package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/woodhead-tech/alertmind/internal/config"
)

func newTestServer() *Server {
	return New(&config.Config{
		AnthropicAPIKey: "test-key",
		Model:           "claude-haiku-4-5-20251001",
		Port:            "8080",
		FetchRunbooks:   false,
	})
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	newTestServer().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got: %v", body)
	}
}

func TestWebhook_validPayload(t *testing.T) {
	payload := `{
		"version": "4",
		"status": "firing",
		"receiver": "alertmind",
		"commonLabels": {"alertname": "TestAlert", "severity": "warning"},
		"commonAnnotations": {},
		"alerts": [{"status": "firing", "labels": {"alertname": "TestAlert"}, "annotations": {}}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	req.Header.Set("content-type", "application/json")
	w := httptest.NewRecorder()
	newTestServer().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWebhook_invalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	newTestServer().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTest_returnsAccepted(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	newTestServer().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}
