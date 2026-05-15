// Package notifier implements alert delivery to Slack and Discord.
package notifier

import (
	"context"

	"github.com/woodhead-tech/alertmind/internal/alert"
)

// Notifier sends a triage summary for an alert group.
type Notifier interface {
	Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error
}

// Multi fans out to all configured notifiers and returns the last error, if any.
// It always calls every notifier — a failure from one does not skip the rest.
type Multi struct {
	notifiers []Notifier
}

// NewMulti returns a Multi that delivers to all provided notifiers.
func NewMulti(notifiers ...Notifier) *Multi {
	return &Multi{notifiers: notifiers}
}

// Notify calls every notifier and returns the last error encountered.
func (m *Multi) Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error {
	var last error
	for _, n := range m.notifiers {
		if err := n.Notify(ctx, payload, triage); err != nil {
			last = err
		}
	}
	return last
}
