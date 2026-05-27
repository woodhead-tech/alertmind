package notifier

import (
	"context"
	"errors"
	"testing"

	"github.com/woodhead-tech/alertmind/internal/alert"
)

type stubNotifier struct {
	called bool
	err    error
}

func (s *stubNotifier) Notify(_ context.Context, _ *alert.AlertmanagerPayload, _ *alert.Triage) error {
	s.called = true
	return s.err
}

func TestMulti_callsAllNotifiers(t *testing.T) {
	a := &stubNotifier{}
	b := &stubNotifier{}
	m := NewMulti(a, b)

	m.Notify(context.Background(), &alert.AlertmanagerPayload{}, &alert.Triage{})

	if !a.called {
		t.Error("first notifier not called")
	}
	if !b.called {
		t.Error("second notifier not called")
	}
}

func TestMulti_returnsErrorOnlyWhenAllFail(t *testing.T) {
	errA := errors.New("notifier A failed")
	errB := errors.New("notifier B failed")
	m := NewMulti(&stubNotifier{err: errA}, &stubNotifier{err: errB})

	err := m.Notify(context.Background(), &alert.AlertmanagerPayload{}, &alert.Triage{})

	if err == nil {
		t.Error("expected error when all notifiers fail, got nil")
	}
	if !errors.Is(err, errA) {
		t.Errorf("expected joined error to contain errA, got: %v", err)
	}
	if !errors.Is(err, errB) {
		t.Errorf("expected joined error to contain errB, got: %v", err)
	}
}

func TestMulti_returnsNilWhenAtLeastOneSucceeds(t *testing.T) {
	errA := errors.New("notifier A failed")
	m := NewMulti(&stubNotifier{err: errA}, &stubNotifier{})

	err := m.Notify(context.Background(), &alert.AlertmanagerPayload{}, &alert.Triage{})

	if err != nil {
		t.Errorf("expected nil when at least one notifier succeeds, got: %v", err)
	}
}

func TestMulti_continuesAfterFirstError(t *testing.T) {
	b := &stubNotifier{}
	m := NewMulti(&stubNotifier{err: errors.New("failed")}, b)

	m.Notify(context.Background(), &alert.AlertmanagerPayload{}, &alert.Triage{})

	if !b.called {
		t.Error("second notifier should be called even if first fails")
	}
}

func TestMulti_noNotifiers(t *testing.T) {
	m := NewMulti()
	err := m.Notify(context.Background(), &alert.AlertmanagerPayload{}, &alert.Triage{})
	if err != nil {
		t.Errorf("expected nil error for empty notifier list, got: %v", err)
	}
}
