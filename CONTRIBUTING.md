# Contributing to alertmind

Bug reports, fixes, and features are welcome.

## Getting started

```bash
git clone https://github.com/whitenhiemer/alertmind
cd alertmind
go build ./...
go test ./...
```

No external dependencies — stdlib only. Everything should build and test clean out of the box.

## Running locally

```bash
cp .env.example .env
# Edit .env with your ANTHROPIC_API_KEY and a webhook URL

source .env
go run . 
```

Fire a synthetic alert to test the full pipeline:

```bash
curl -X POST http://localhost:8080/test
```

## Project structure

```
main.go                   # HTTP server startup, signal handling
internal/
  alert/types.go          # Alertmanager webhook payload types
  config/config.go        # Env var config loading
  enricher/enricher.go    # Builds LLM prompt from alert payload
  llm/client.go           # Anthropic API client (direct HTTP)
  notifier/               # Slack + Discord output formatters
  server/server.go        # HTTP route handlers
```

## Making changes

- **Keep it stdlib-only.** No external Go dependencies. The Anthropic API is called over raw `net/http`.
- **Test coverage.** Each package has a `_test.go`. Add tests for new behavior. Use `httptest` for anything that makes HTTP calls.
- **Non-blocking webhook handler.** The `/webhook` handler must respond 200 immediately — triage work runs in a goroutine. Don't block the response path.

## Submitting a PR

1. Fork the repo and create a branch from `main`
2. Make your changes with tests
3. Run `go test -race ./...` — all tests must pass
4. Open a PR with a clear description of what and why

## Reporting bugs

Open an issue with:
- alertmind version or commit hash
- Your Alertmanager config (redact webhook URLs)
- The alert payload that caused the issue (if applicable)
- What you expected vs. what happened
