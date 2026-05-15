// Package alert defines the wire types for Alertmanager webhook payloads and LLM triage output.
package alert

import "time"

// AlertmanagerPayload is the webhook payload sent by Alertmanager on each fired or resolved group.
type AlertmanagerPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"` // "firing" | "resolved"
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert is a single alert within a group payload.
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL"`
}

// Triage is the structured analysis returned by the LLM for a firing alert group.
type Triage struct {
	ProbableCause         string   `json:"probable_cause"`
	SeverityAssessment    string   `json:"severity_assessment"`
	ImmediateActions      []string `json:"immediate_actions"`
	InvestigationCommands []string `json:"investigation_commands"`
	Notes                 string   `json:"notes,omitempty"`
}
