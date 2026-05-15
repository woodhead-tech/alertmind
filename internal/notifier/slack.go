package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/whitenhiemer/alertmind/internal/alert"
)

// Slack delivers triage summaries as Slack Block Kit messages via an incoming webhook.
type Slack struct {
	webhookURL string
	http       *http.Client
}

// NewSlack returns a Slack notifier that posts to the given webhook URL.
func NewSlack(webhookURL string) *Slack {
	return &Slack{webhookURL: webhookURL, http: &http.Client{Timeout: 10 * time.Second}}
}

type slackPayload struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string       `json:"type"`
	Text *slackText   `json:"text,omitempty"`
	Elements []slackText `json:"elements,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Notify builds and POSTs a Slack Block Kit message for the alert group.
func (s *Slack) Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error {
	msg := buildSlackMessage(payload, triage)

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}

func buildSlackMessage(payload *alert.AlertmanagerPayload, triage *alert.Triage) slackPayload {
	emoji := "🚨"
	if payload.Status == "resolved" {
		emoji = "✅"
	}

	alertName := payload.CommonLabels["alertname"]
	severity := payload.CommonLabels["severity"]
	instance := payload.CommonLabels["instance"]

	title := fmt.Sprintf("%s *%s* [%s]", emoji, alertName, strings.ToUpper(payload.Status))
	if instance != "" {
		title += fmt.Sprintf(" — %s", instance)
	}

	blocks := []slackBlock{
		{
			Type: "section",
			Text: &slackText{Type: "mrkdwn", Text: title},
		},
		{Type: "divider"},
	}

	if payload.Status == "firing" {
		blocks = append(blocks,
			slackBlock{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Probable Cause*\n%s", triage.ProbableCause)},
			},
			slackBlock{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Severity Assessment*\n%s", triage.SeverityAssessment)},
			},
		)

		if len(triage.ImmediateActions) > 0 {
			actions := make([]string, len(triage.ImmediateActions))
			for i, a := range triage.ImmediateActions {
				actions[i] = fmt.Sprintf("%d. %s", i+1, a)
			}
			blocks = append(blocks, slackBlock{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Immediate Actions*\n%s", strings.Join(actions, "\n"))},
			})
		}

		if len(triage.InvestigationCommands) > 0 {
			cmds := strings.Join(triage.InvestigationCommands, "\n")
			blocks = append(blocks, slackBlock{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Investigation*\n```%s```", cmds)},
			})
		}

		if triage.Notes != "" {
			blocks = append(blocks, slackBlock{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: fmt.Sprintf("_Note: %s_", triage.Notes)},
			})
		}
	}

	meta := fmt.Sprintf("alertmind • %s", severity)
	if len(payload.Alerts) > 0 && !payload.Alerts[0].StartsAt.IsZero() {
		duration := time.Since(payload.Alerts[0].StartsAt).Round(time.Second)
		meta += fmt.Sprintf(" • firing for %s", duration)
	}

	blocks = append(blocks,
		slackBlock{Type: "divider"},
		slackBlock{
			Type:     "context",
			Elements: []slackText{{Type: "mrkdwn", Text: meta}},
		},
	)

	return slackPayload{Blocks: blocks}
}
