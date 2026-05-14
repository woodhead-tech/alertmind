package notifier

import (
	"strings"
	"testing"
	"time"

	"github.com/whitenhiemer/alertmind/internal/alert"
)

func slackTestPayload(status string) *alert.AlertmanagerPayload {
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

func slackTestTriage() *alert.Triage {
	return &alert.Triage{
		ProbableCause:         "Disk filled by log files.",
		SeverityAssessment:    "High. Node will OOM soon.",
		ImmediateActions:      []string{"run du -sh /*", "clear /var/log"},
		InvestigationCommands: []string{"df -h", "du -sh /var/log/*"},
		Notes:                 "Check logrotate config.",
	}
}

func collectSlackText(msg slackPayload) string {
	var sb strings.Builder
	for _, b := range msg.Blocks {
		if b.Text != nil {
			sb.WriteString(b.Text.Text)
		}
		for _, e := range b.Elements {
			sb.WriteString(e.Text)
		}
	}
	return sb.String()
}

func TestBuildSlackMessage_firingContainsTriageFields(t *testing.T) {
	msg := buildSlackMessage(slackTestPayload("firing"), slackTestTriage())
	combined := collectSlackText(msg)

	for _, want := range []string{"DiskFull", "FIRING", "Disk filled by log files", "df -h", "logrotate"} {
		if !strings.Contains(combined, want) {
			t.Errorf("slack message missing %q", want)
		}
	}
}

func TestBuildSlackMessage_resolvedOmitsTriageFields(t *testing.T) {
	msg := buildSlackMessage(slackTestPayload("resolved"), slackTestTriage())
	combined := collectSlackText(msg)

	if strings.Contains(combined, "Probable Cause") {
		t.Error("resolved message should not contain triage fields")
	}
	if !strings.Contains(combined, "RESOLVED") {
		t.Error("resolved message should indicate RESOLVED status")
	}
}

func TestBuildSlackMessage_numbersActions(t *testing.T) {
	msg := buildSlackMessage(slackTestPayload("firing"), slackTestTriage())
	combined := collectSlackText(msg)

	if !strings.Contains(combined, "1. run du") {
		t.Error("expected numbered immediate actions")
	}
}
