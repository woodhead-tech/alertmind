package notifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/woodhead-tech/alertmind/internal/alert"
)

func TestWhatsApp_Notify(t *testing.T) {
	// Mock Twilio API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify basic auth
		username, password, ok := r.BasicAuth()
		if !ok || username != "sid" || password != "token" {
			t.Errorf("expected basic auth sid:token, got %s:%s", username, password)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		// Verify body
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}
		if r.FormValue("From") != "whatsapp:+123456789" {
			t.Errorf("expected From whatsapp:+123456789, got %s", r.FormValue("From"))
		}
		if r.FormValue("To") != "whatsapp:+987654321" {
			t.Errorf("expected To whatsapp:+987654321, got %s", r.FormValue("To"))
		}
		if !strings.Contains(r.FormValue("Body"), "DiskFull") {
			t.Errorf("expected Body to contain DiskFull, got %s", r.FormValue("Body"))
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sid": "SMxxx"}`))
	}))
	defer server.Close()

	// To test against the mock server, we need to bypass the hardcoded Twilio URL.
	// Since we can't easily change the hardcoded URL without refactoring,
	// let's refactor WhatsApp struct to take an optional base URL.
	
	w := &WhatsApp{
		accountSID: "sid",
		authToken:  "token",
		from:       "whatsapp:+123456789",
		to:         "whatsapp:+987654321",
		baseURL:    server.URL,
		http:       server.Client(),
	}

	err := w.Notify(context.Background(), whatsappTestPayload("firing"), whatsappTestTriage())
	if err != nil {
		t.Errorf("Notify failed: %v", err)
	}
}

func whatsappTestPayload(status string) *alert.AlertmanagerPayload {
	return &alert.AlertmanagerPayload{
		Status: status,
		CommonLabels: map[string]string{
			"alertname": "DiskFull",
			"severity":  "critical",
			"instance":  "node2:9100",
		},
		Alerts: []alert.Alert{
			{Status: status, StartsAt: time.Now().Add(-10 * time.Minute)},
		},
	}
}

func whatsappTestTriage() *alert.Triage {
	return &alert.Triage{
		ProbableCause:         "Disk filled by log files.",
		SeverityAssessment:    "High. Node will OOM soon.",
		ImmediateActions:      []string{"run du -sh /*", "clear /var/log"},
		InvestigationCommands: []string{"df -h", "du -sh /var/log/*"},
		Notes:                 "Check logrotate config.",
	}
}

func TestBuildWhatsAppMessage_firingContainsTriageFields(t *testing.T) {
	msg := buildWhatsAppMessage(whatsappTestPayload("firing"), whatsappTestTriage())

	expectedStrings := []string{
		"🚨",
		"DiskFull",
		"FIRING",
		"Probable Cause",
		"Disk filled by log files",
		"Immediate Actions",
		"1. run du -sh /*",
		"Investigation",
		"df -h",
		"Note: Check logrotate config.",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(msg, want) {
			t.Errorf("whatsapp message missing %q", want)
		}
	}
}

func TestBuildWhatsAppMessage_resolvedOmitsTriageFields(t *testing.T) {
	msg := buildWhatsAppMessage(whatsappTestPayload("resolved"), whatsappTestTriage())

	if strings.Contains(msg, "Probable Cause") {
		t.Error("resolved message should not contain triage fields")
	}
	if !strings.Contains(msg, "RESOLVED") {
		t.Error("resolved message should indicate RESOLVED status")
	}
	if !strings.Contains(msg, "✅") {
		t.Error("resolved message should contain checkmark emoji")
	}
}

func TestNewWhatsApp_PrefixHandling(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		wantFrom string
		wantTo   string
	}{
		{
			name:     "no prefixes",
			from:     "+123456789",
			to:       "+987654321",
			wantFrom: "whatsapp:+123456789",
			wantTo:   "whatsapp:+987654321",
		},
		{
			name:     "with prefixes",
			from:     "whatsapp:+123456789",
			to:       "whatsapp:+987654321",
			wantFrom: "whatsapp:+123456789",
			wantTo:   "whatsapp:+987654321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWhatsApp("sid", "token", tt.from, tt.to)
			if w.from != tt.wantFrom {
				t.Errorf("got from %q, want %q", w.from, tt.wantFrom)
			}
			if w.to != tt.wantTo {
				t.Errorf("got to %q, want %q", w.to, tt.wantTo)
			}
		})
	}
}
