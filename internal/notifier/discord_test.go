package notifier

import (
	"strings"
	"testing"
	"time"

	"github.com/woodhead-tech/alertmind/internal/alert"
)

func discordTestPayload(status, severity string) *alert.AlertmanagerPayload {
	return &alert.AlertmanagerPayload{
		Status: status,
		CommonLabels: map[string]string{
			"alertname": "HighMemory",
			"severity":  severity,
			"instance":  "node3:9100",
		},
		Alerts: []alert.Alert{
			{Status: status, StartsAt: time.Now().Add(-3 * time.Minute)},
		},
	}
}

func discordTestTriage() *alert.Triage {
	return &alert.Triage{
		ProbableCause:         "Memory leak in app.",
		SeverityAssessment:    "High.",
		ImmediateActions:      []string{"restart pod"},
		InvestigationCommands: []string{"kubectl top pod"},
	}
}

func TestBuildDiscordMessage_colorCriticalFiring(t *testing.T) {
	msg := buildDiscordMessage(discordTestPayload("firing", "critical"), discordTestTriage())

	if msg.Embeds[0].Color != colorFiring {
		t.Errorf("expected colorFiring %d, got %d", colorFiring, msg.Embeds[0].Color)
	}
}

func TestBuildDiscordMessage_colorWarningFiring(t *testing.T) {
	msg := buildDiscordMessage(discordTestPayload("firing", "warning"), discordTestTriage())

	if msg.Embeds[0].Color != colorWarning {
		t.Errorf("expected colorWarning %d, got %d", colorWarning, msg.Embeds[0].Color)
	}
}

func TestBuildDiscordMessage_colorResolved(t *testing.T) {
	msg := buildDiscordMessage(discordTestPayload("resolved", "warning"), discordTestTriage())

	if msg.Embeds[0].Color != colorResolved {
		t.Errorf("expected colorResolved %d, got %d", colorResolved, msg.Embeds[0].Color)
	}
}

func TestBuildDiscordMessage_commandsTruncatedAtLimit(t *testing.T) {
	triage := &alert.Triage{
		ProbableCause:         "test",
		SeverityAssessment:    "test",
		InvestigationCommands: []string{strings.Repeat("x", 1000)},
	}

	msg := buildDiscordMessage(discordTestPayload("firing", "critical"), triage)

	var cmdField *discordField
	for i := range msg.Embeds[0].Fields {
		if msg.Embeds[0].Fields[i].Name == "Investigation Commands" {
			cmdField = &msg.Embeds[0].Fields[i]
			break
		}
	}
	if cmdField == nil {
		t.Fatal("expected Investigation Commands field")
	}
	if len(cmdField.Value) > 1024 {
		t.Errorf("field value exceeds Discord 1024 char limit: %d chars", len(cmdField.Value))
	}
}

func TestBuildDiscordMessage_resolvedHasNoTriageFields(t *testing.T) {
	msg := buildDiscordMessage(discordTestPayload("resolved", "warning"), discordTestTriage())

	if len(msg.Embeds[0].Fields) > 0 {
		t.Error("resolved message should have no triage fields")
	}
}
