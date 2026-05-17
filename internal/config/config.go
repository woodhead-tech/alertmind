// Package config loads alertmind's runtime configuration from environment variables.
package config

import "os"

// Config holds all runtime settings. Populate via Load().
type Config struct {
	AnthropicAPIKey   string
	SlackWebhookURL   string
	DiscordWebhookURL string
	TwilioAccountSID  string
	TwilioAuthToken   string
	WhatsAppFrom      string
	WhatsAppTo        string
	Port              string
	Model             string
	FetchRunbooks     bool
}

// Load reads configuration from environment variables and applies defaults.
// At least ANTHROPIC_API_KEY and one of SLACK_WEBHOOK_URL or DISCORD_WEBHOOK_URL must be set.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	model := os.Getenv("ALERTMIND_MODEL")
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	return &Config{
		AnthropicAPIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		SlackWebhookURL:   os.Getenv("SLACK_WEBHOOK_URL"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		TwilioAccountSID:  os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:   os.Getenv("TWILIO_AUTH_TOKEN"),
		WhatsAppFrom:      os.Getenv("WHATSAPP_FROM"),
		WhatsAppTo:        os.Getenv("WHATSAPP_TO"),
		Port:              port,
		Model:             model,
		// FetchRunbooks defaults to true; set FETCH_RUNBOOKS=false to disable.
		FetchRunbooks: os.Getenv("FETCH_RUNBOOKS") != "false",
	}
}
