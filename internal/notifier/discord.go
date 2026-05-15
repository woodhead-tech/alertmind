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

const (
	colorFiring   = 0xE53935 // red
	colorResolved = 0x43A047 // green
	colorWarning  = 0xFB8C00 // orange
)

// Discord delivers triage summaries as Discord embeds via an incoming webhook.
type Discord struct {
	webhookURL string
	http       *http.Client
}

// NewDiscord returns a Discord notifier that posts to the given webhook URL.
func NewDiscord(webhookURL string) *Discord {
	return &Discord{webhookURL: webhookURL, http: &http.Client{Timeout: 10 * time.Second}}
}

type discordPayload struct {
	Username string         `json:"username"`
	Embeds   []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text"`
}

// Notify builds and POSTs a Discord embed for the alert group.
func (d *Discord) Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error {
	msg := buildDiscordMessage(payload, triage)

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned %d", resp.StatusCode)
	}
	return nil
}

func buildDiscordMessage(payload *alert.AlertmanagerPayload, triage *alert.Triage) discordPayload {
	alertName := payload.CommonLabels["alertname"]
	severity := payload.CommonLabels["severity"]
	instance := payload.CommonLabels["instance"]

	emoji := "🚨"
	color := colorFiring
	if payload.Status == "resolved" {
		emoji = "✅"
		color = colorResolved
	} else if severity == "warning" {
		color = colorWarning
	}

	title := fmt.Sprintf("%s %s [%s]", emoji, alertName, strings.ToUpper(payload.Status))
	if instance != "" {
		title += fmt.Sprintf(" — %s", instance)
	}

	var fields []discordField

	if payload.Status == "firing" {
		fields = append(fields,
			discordField{
				Name:  "Probable Cause",
				Value: triage.ProbableCause,
			},
			discordField{
				Name:  "Severity Assessment",
				Value: triage.SeverityAssessment,
			},
		)

		if len(triage.ImmediateActions) > 0 {
			actions := make([]string, len(triage.ImmediateActions))
			for i, a := range triage.ImmediateActions {
				actions[i] = fmt.Sprintf("%d. %s", i+1, a)
			}
			fields = append(fields, discordField{
				Name:  "Immediate Actions",
				Value: strings.Join(actions, "\n"),
			})
		}

		if len(triage.InvestigationCommands) > 0 {
			cmds := strings.Join(triage.InvestigationCommands, "\n")
			// Discord has a 1024 char field limit
			if len(cmds) > 990 {
				cmds = cmds[:990] + "..."
			}
			fields = append(fields, discordField{
				Name:  "Investigation Commands",
				Value: fmt.Sprintf("```\n%s\n```", cmds),
			})
		}

		if triage.Notes != "" {
			fields = append(fields, discordField{
				Name:  "Notes",
				Value: triage.Notes,
			})
		}
	}

	footer := fmt.Sprintf("alertmind • %s", severity)
	if len(payload.Alerts) > 0 && !payload.Alerts[0].StartsAt.IsZero() {
		duration := time.Since(payload.Alerts[0].StartsAt).Round(time.Second)
		footer += fmt.Sprintf(" • firing for %s", duration)
	}

	return discordPayload{
		Username: "alertmind",
		Embeds: []discordEmbed{
			{
				Title:  title,
				Color:  color,
				Fields: fields,
				Footer: &discordFooter{Text: footer},
			},
		},
	}
}
