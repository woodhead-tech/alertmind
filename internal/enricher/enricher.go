// Package enricher builds the LLM prompt from an Alertmanager payload,
// optionally fetching runbook content to include as context.
package enricher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/whitenhiemer/alertmind/internal/alert"
)

const (
	// runbookMaxBytes caps how much runbook content is included in the prompt.
	// Keeps context size predictable and avoids blowing the model's token budget.
	runbookMaxBytes = 3000
	fetchTimeout    = 5 * time.Second
)

// BuildPrompt constructs the LLM user prompt from an alert payload and optional runbook content.
// The resulting string is passed directly to the LLM as the user turn.
func BuildPrompt(payload *alert.AlertmanagerPayload, fetchRunbooks bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Status: %s\n", strings.ToUpper(payload.Status)))
	sb.WriteString(fmt.Sprintf("Alerts in group: %d\n\n", len(payload.Alerts)))

	for i, a := range payload.Alerts {
		sb.WriteString(fmt.Sprintf("--- Alert %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("Name:     %s\n", a.Labels["alertname"]))
		sb.WriteString(fmt.Sprintf("Severity: %s\n", a.Labels["severity"]))
		if v := a.Labels["instance"]; v != "" {
			sb.WriteString(fmt.Sprintf("Instance: %s\n", v))
		}
		if v := a.Labels["job"]; v != "" {
			sb.WriteString(fmt.Sprintf("Job:      %s\n", v))
		}
		if v := a.Annotations["summary"]; v != "" {
			sb.WriteString(fmt.Sprintf("Summary:  %s\n", v))
		}
		if v := a.Annotations["description"]; v != "" {
			sb.WriteString(fmt.Sprintf("Details:  %s\n", v))
		}

		if !a.StartsAt.IsZero() {
			duration := time.Since(a.StartsAt).Round(time.Second)
			sb.WriteString(fmt.Sprintf("Firing for: %s\n", duration))
		}

		// Append any labels not already covered above.
		skip := map[string]bool{"alertname": true, "severity": true, "instance": true, "job": true}
		for k, v := range a.Labels {
			if !skip[k] {
				sb.WriteString(fmt.Sprintf("Label %s=%s\n", k, v))
			}
		}

		if fetchRunbooks {
			if url := a.Annotations["runbook_url"]; url != "" {
				if content := fetchRunbook(url); content != "" {
					sb.WriteString(fmt.Sprintf("\nRunbook (%s):\n%s\n", url, content))
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// fetchRunbook fetches the runbook at url and returns its content, capped at runbookMaxBytes.
// Returns an empty string on any error so callers can skip silently.
func fetchRunbook(url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, runbookMaxBytes))
	if err != nil {
		return ""
	}
	return string(body)
}
