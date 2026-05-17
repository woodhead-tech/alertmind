// Package server implements the alertmind HTTP server.
// Routes: GET /health, POST /webhook (Alertmanager), POST /test (synthetic alert).
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/woodhead-tech/alertmind/internal/alert"
	"github.com/woodhead-tech/alertmind/internal/config"
	"github.com/woodhead-tech/alertmind/internal/enricher"
	"github.com/woodhead-tech/alertmind/internal/llm"
	"github.com/woodhead-tech/alertmind/internal/notifier"
)

// Server wires together the LLM client, notifiers, and HTTP routes.
type Server struct {
	cfg      *config.Config
	llm      *llm.Client
	notifier notifier.Notifier
	mux      *http.ServeMux
}

// New creates a Server from cfg, initialising notifiers based on which webhook URLs are set.
func New(cfg *config.Config) *Server {
	llmClient := llm.New(cfg.AnthropicAPIKey, cfg.Model)

	var notifiers []notifier.Notifier
	if cfg.SlackWebhookURL != "" {
		notifiers = append(notifiers, notifier.NewSlack(cfg.SlackWebhookURL))
		log.Println("notifier: Slack enabled")
	}
	if cfg.DiscordWebhookURL != "" {
		notifiers = append(notifiers, notifier.NewDiscord(cfg.DiscordWebhookURL))
		log.Println("notifier: Discord enabled")
	}
	if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" && cfg.WhatsAppFrom != "" && cfg.WhatsAppTo != "" {
		notifiers = append(notifiers, notifier.NewWhatsApp(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.WhatsAppFrom, cfg.WhatsAppTo))
		log.Println("notifier: WhatsApp enabled")
	}
	if len(notifiers) == 0 {
		log.Println("warning: no notifiers configured (set SLACK_WEBHOOK_URL or DISCORD_WEBHOOK_URL)")
	}

	s := &Server{
		cfg:      cfg,
		llm:      llmClient,
		notifier: notifier.NewMulti(notifiers...),
		mux:      http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("POST /webhook", s.webhook)
	s.mux.HandleFunc("POST /test", s.test)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Start begins listening on the configured port. Blocks until the server exits.
func (s *Server) Start() error {
	return http.ListenAndServe(":"+s.cfg.Port, s.mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	var payload alert.AlertmanagerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Respond immediately so Alertmanager doesn't time out.
	w.WriteHeader(http.StatusOK)

	go s.process(&payload)
}

func (s *Server) test(w http.ResponseWriter, r *http.Request) {
	payload := testPayload()
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "processing test alert"})
	go s.process(payload)
}

// process runs the full triage pipeline in a goroutine: enrich → LLM → notify.
// On LLM failure it sends a fallback notification rather than dropping the alert silently.
func (s *Server) process(payload *alert.AlertmanagerPayload) {
	if len(payload.Alerts) == 0 {
		return
	}

	alertName := payload.CommonLabels["alertname"]
	log.Printf("processing %s alert group (%d alerts, status=%s)", alertName, len(payload.Alerts), payload.Status)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alertContext := enricher.BuildPrompt(payload, s.cfg.FetchRunbooks)

	triage, err := s.llm.Triage(ctx, alertContext)
	if err != nil {
		log.Printf("triage error for %s: %v", alertName, err)
		// Send a fallback notification so the alert isn't silently dropped.
		triage = &alert.Triage{
			ProbableCause:      "alertmind could not generate triage — check logs.",
			SeverityAssessment: payload.CommonLabels["severity"],
		}
	} else {
		log.Printf("triage for %s: cause=%q actions=%d commands=%d",
			alertName, triage.ProbableCause, len(triage.ImmediateActions), len(triage.InvestigationCommands))
	}

	if err := s.notifier.Notify(ctx, payload, triage); err != nil {
		log.Printf("notify error for %s: %v", alertName, err)
		return
	}

	log.Printf("triage sent for %s", alertName)
}

func testPayload() *alert.AlertmanagerPayload {
	return &alert.AlertmanagerPayload{
		Version:  "4",
		Status:   "firing",
		Receiver: "alertmind",
		CommonLabels: map[string]string{
			"alertname": "HighCPUUsage",
			"severity":  "warning",
			"instance":  "node1:9100",
			"job":       "node-exporter",
		},
		CommonAnnotations: map[string]string{
			"summary":     "CPU usage above 90% on node1",
			"description": "CPU usage has been above 90% for more than 5 minutes on node1:9100.",
		},
		Alerts: []alert.Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPUUsage",
					"severity":  "warning",
					"instance":  "node1:9100",
					"job":       "node-exporter",
				},
				Annotations: map[string]string{
					"summary":     "CPU usage above 90% on node1",
					"description": "CPU usage has been above 90% for more than 5 minutes on node1:9100.",
				},
				StartsAt:    time.Now().Add(-6 * time.Minute),
				Fingerprint: "abc123def456",
			},
		},
	}
}
