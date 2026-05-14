package enricher

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/whitenhiemer/alertmind/internal/alert"
)

func testPayload() *alert.AlertmanagerPayload {
	return &alert.AlertmanagerPayload{
		Status: "firing",
		CommonLabels: map[string]string{
			"alertname": "HighCPU",
			"severity":  "warning",
		},
		Alerts: []alert.Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPU",
					"severity":  "warning",
					"instance":  "node1:9100",
					"job":       "node-exporter",
				},
				Annotations: map[string]string{
					"summary":     "CPU high",
					"description": "CPU above 90%",
				},
				StartsAt: time.Now().Add(-5 * time.Minute),
			},
		},
	}
}

func TestBuildPrompt_containsAlertFields(t *testing.T) {
	prompt := BuildPrompt(testPayload(), false)

	for _, want := range []string{"FIRING", "HighCPU", "warning", "node1:9100", "CPU high", "CPU above 90%"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPrompt_noRunbooksWhenDisabled(t *testing.T) {
	payload := testPayload()
	payload.Alerts[0].Annotations["runbook_url"] = "http://example.com/runbook"

	prompt := BuildPrompt(payload, false)

	if strings.Contains(prompt, "Runbook") {
		t.Error("expected no runbook content when fetchRunbooks=false")
	}
}

func TestBuildPrompt_fetchesRunbook(t *testing.T) {
	content := "# Runbook\nCheck the CPU usage with `top`."
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer srv.Close()

	payload := testPayload()
	payload.Alerts[0].Annotations["runbook_url"] = srv.URL

	prompt := BuildPrompt(payload, true)

	if !strings.Contains(prompt, "Runbook") {
		t.Error("expected runbook section in prompt")
	}
	if !strings.Contains(prompt, content) {
		t.Error("expected runbook content in prompt")
	}
}

func TestBuildPrompt_runbookFetchFail_stillIncludesAlert(t *testing.T) {
	payload := testPayload()
	// Point at a port nothing is listening on.
	payload.Alerts[0].Annotations["runbook_url"] = "http://127.0.0.1:1/nonexistent"

	prompt := BuildPrompt(payload, true)

	if !strings.Contains(prompt, "HighCPU") {
		t.Error("expected alert content even when runbook fetch fails")
	}
}
