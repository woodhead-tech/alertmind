package notifier

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/woodhead-tech/alertmind/internal/alert"
)

// WhatsApp delivers triage summaries as WhatsApp messages via Twilio.
type WhatsApp struct {
	accountSID string
	authToken  string
	from       string
	to         string
	baseURL    string // For testing
	http       *http.Client
}

// NewWhatsApp returns a WhatsApp notifier that sends messages via Twilio.
func NewWhatsApp(accountSID, authToken, from, to string) *WhatsApp {
	if !strings.HasPrefix(from, "whatsapp:") {
		from = "whatsapp:" + from
	}
	if !strings.HasPrefix(to, "whatsapp:") {
		to = "whatsapp:" + to
	}
	return &WhatsApp{
		accountSID: accountSID,
		authToken:  authToken,
		from:       from,
		to:         to,
		baseURL:    "https://api.twilio.com",
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify builds and sends a WhatsApp message for the alert group.
func (w *WhatsApp) Notify(ctx context.Context, payload *alert.AlertmanagerPayload, triage *alert.Triage) error {
	msg := buildWhatsAppMessage(payload, triage)

	apiURL := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", w.baseURL, w.accountSID)

	data := url.Values{}
	data.Set("From", w.from)
	data.Set("To", w.to)
	data.Set("Body", msg)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(w.accountSID, w.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("twilio API returned %d", resp.StatusCode)
	}

	return nil
}

func buildWhatsAppMessage(payload *alert.AlertmanagerPayload, triage *alert.Triage) string {
	emoji := "🚨"
	if payload.Status == "resolved" {
		emoji = "✅"
	}

	alertName := payload.CommonLabels["alertname"]
	status := strings.ToUpper(payload.Status)

	var b strings.Builder
	fmt.Fprintf(&b, "%s *%s* [%s]\n", emoji, alertName, status)

	if payload.Status == "firing" {
		fmt.Fprintf(&b, "\n*Probable Cause*\n%s\n", triage.ProbableCause)

		if len(triage.ImmediateActions) > 0 {
			fmt.Fprint(&b, "\n*Immediate Actions*\n")
			for i, a := range triage.ImmediateActions {
				fmt.Fprintf(&b, "%d. %s\n", i+1, a)
			}
		}

		if len(triage.InvestigationCommands) > 0 {
			fmt.Fprint(&b, "\n*Investigation*\n")
			for _, cmd := range triage.InvestigationCommands {
				fmt.Fprintf(&b, "```%s```\n", cmd)
			}
		}
	}

	if triage.Notes != "" {
		fmt.Fprintf(&b, "\n_Note: %s_\n", triage.Notes)
	}

	return b.String()
}
