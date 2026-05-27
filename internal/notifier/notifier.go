// Package notifier implements alert delivery to Slack and Discord.
package notifier

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/woodhead-tech/alertmind/internal/alert"
)

// Notifier sends a triage summary for an alert group.
type Notifier interface {
	Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error
}

// Multi fans out to all configured notifiers and returns an error only if
// every notifier fails. A partial failure is logged but not returned.
type Multi struct {
	notifiers []Notifier
}

// NewMulti returns a Multi that delivers to all provided notifiers.
func NewMulti(notifiers ...Notifier) *Multi {
	return &Multi{notifiers: notifiers}
}

// Notify calls every notifier. It logs per-notifier errors and returns an
// error only when all notifiers fail.
func (m *Multi) Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error {
	var errs []error
	ok := 0
	for _, n := range m.notifiers {
		name := reflect.TypeOf(n).Elem().Name()
		if err := n.Notify(ctx, payload, triage); err != nil {
			log.Printf("notifier %s error: %v", name, err)
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		} else {
			ok++
		}
	}
	if ok == 0 && len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
