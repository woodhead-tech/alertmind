// alertmind receives Alertmanager webhooks, triages them with an LLM, and posts
// structured summaries to Slack and/or Discord.
//
// Required env: ANTHROPIC_API_KEY
// At least one of: SLACK_WEBHOOK_URL, DISCORD_WEBHOOK_URL
// See README for full configuration reference.
package main

import (
	"log"

	"github.com/woodhead-tech/alertmind/internal/config"
	"github.com/woodhead-tech/alertmind/internal/server"
)

func main() {
	cfg := config.Load()

	if cfg.AnthropicAPIKey == "" {
		log.Fatal("ANTHROPIC_API_KEY is required")
	}

	log.Printf("alertmind starting on :%s (model=%s fetch_runbooks=%v)",
		cfg.Port, cfg.Model, cfg.FetchRunbooks)

	srv := server.New(cfg)
	log.Fatal(srv.Start())
}
