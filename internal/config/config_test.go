package config

import (
	"testing"
)

func TestLoad_defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("ALERTMIND_MODEL", "")
	t.Setenv("FETCH_RUNBOOKS", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("expected default model, got %s", cfg.Model)
	}
	if !cfg.FetchRunbooks {
		t.Error("expected FetchRunbooks=true by default")
	}
}

func TestLoad_envOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("ALERTMIND_MODEL", "claude-opus-4-7")
	t.Setenv("FETCH_RUNBOOKS", "false")
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/test")
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/test")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.Model != "claude-opus-4-7" {
		t.Errorf("expected claude-opus-4-7, got %s", cfg.Model)
	}
	if cfg.FetchRunbooks {
		t.Error("expected FetchRunbooks=false")
	}
	if cfg.AnthropicAPIKey != "test-key" {
		t.Errorf("expected api key test-key, got %s", cfg.AnthropicAPIKey)
	}
	if cfg.SlackWebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("unexpected slack url: %s", cfg.SlackWebhookURL)
	}
	if cfg.DiscordWebhookURL != "https://discord.com/api/webhooks/test" {
		t.Errorf("unexpected discord url: %s", cfg.DiscordWebhookURL)
	}
}
