package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractJSON_clean(t *testing.T) {
	in := `{"key": "value"}`
	if got := extractJSON(in); got != in {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestExtractJSON_stripsJsonCodeFence(t *testing.T) {
	in := "```json\n{\"key\": \"value\"}\n```"
	want := `{"key": "value"}`
	if got := extractJSON(in); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExtractJSON_stripsPlainCodeFence(t *testing.T) {
	in := "```\n{\"key\": \"value\"}\n```"
	want := `{"key": "value"}`
	if got := extractJSON(in); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestTriage_success(t *testing.T) {
	responseText := `{
		"probable_cause": "high load",
		"severity_assessment": "critical",
		"immediate_actions": ["check top", "reduce load"],
		"investigation_commands": ["uptime", "htop"],
		"notes": ""
	}`

	apiResp := messagesResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{Type: "text", Text: responseText},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version header")
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(apiResp)
	}))
	defer srv.Close()

	client := newWithURL("test-key", "claude-haiku-4-5-20251001", srv.URL)
	triage, err := client.Triage(context.Background(), "test alert context")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triage.ProbableCause != "high load" {
		t.Errorf("unexpected probable cause: %s", triage.ProbableCause)
	}
	if len(triage.ImmediateActions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(triage.ImmediateActions))
	}
}

func TestTriage_apiError(t *testing.T) {
	apiResp := messagesResponse{
		Error: &struct {
			Message string `json:"message"`
		}{Message: "invalid api key"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(apiResp)
	}))
	defer srv.Close()

	client := newWithURL("bad-key", "claude-haiku-4-5-20251001", srv.URL)
	_, err := client.Triage(context.Background(), "test")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("expected API error message in: %v", err)
	}
}

func TestTriage_emptyContent(t *testing.T) {
	apiResp := messagesResponse{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResp)
	}))
	defer srv.Close()

	client := newWithURL("test-key", "claude-haiku-4-5-20251001", srv.URL)
	_, err := client.Triage(context.Background(), "test")

	if err == nil {
		t.Fatal("expected error for empty content")
	}
}
