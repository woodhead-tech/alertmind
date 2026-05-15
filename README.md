# alertmind

AI-powered alert triage for Alertmanager. When an alert fires, alertmind enriches it
with context, calls Claude to generate structured analysis, and posts a triage summary
to Slack and/or Discord — before your engineer opens their laptop.

```
Alertmanager fires
      │
      ▼
alertmind receives webhook
      │
      ├── fetches runbook content (if annotation URL present)
      │
      ▼
Claude API (claude-haiku — fast, cheap)
      │
      ▼
structured triage:
  • probable cause
  • severity assessment
  • immediate actions (ordered)
  • diagnostic commands
      │
      ├──▶ Slack (Block Kit)
      └──▶ Discord (embed)
```

## Why

Raw alert payloads are noise. Your on-call engineer gets paged at 2am with:

```
[FIRING] HighMemoryUsage on node1 (severity=critical)
```

alertmind turns that into:

> **Probable Cause**
> Memory pressure on node1 is consistent with a Go service leak — RSS has grown
> steadily over 3 hours without a corresponding increase in active connections.
>
> **Severity Assessment**
> Critical — OOM kill is imminent. If node1 runs the API gateway, expect 503s
> within the next 10–20 minutes without intervention.
>
> **Immediate Actions**
> 1. Identify the leaking process: `ps aux --sort=-%mem | head -10`
> 2. Check if the process has grown since last deploy: compare RSS to baseline
> 3. Rolling restart the suspect service if safe to do so
> 4. If RSS is not reclaimed after restart, escalate to a memory dump
>
> **Investigation Commands**
> ```
> ps aux --sort=-%mem | head -10
> cat /proc/$(pgrep api-server)/status | grep VmRSS
> journalctl -u api-server --since "3 hours ago" | grep -i memory
> ```

## Usage

### Docker

```bash
docker run -p 8080:8080 \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  -e DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/... \
  ghcr.io/woodhead-tech/alertmind:latest
```

### Binary

```bash
git clone https://github.com/woodhead-tech/alertmind
cd alertmind
go build -o alertmind .

ANTHROPIC_API_KEY=sk-ant-... \
SLACK_WEBHOOK_URL=https://hooks.slack.com/... \
./alertmind
```

### Configuration

All configuration is via environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `ANTHROPIC_API_KEY` | Yes | — | Anthropic API key |
| `SLACK_WEBHOOK_URL` | No | — | Slack incoming webhook URL |
| `DISCORD_WEBHOOK_URL` | No | — | Discord webhook URL |
| `PORT` | No | `8080` | HTTP listen port |
| `ALERTMIND_MODEL` | No | `claude-haiku-4-5-20251001` | Claude model to use |
| `FETCH_RUNBOOKS` | No | `true` | Fetch runbook URLs from alert annotations |

At least one of `SLACK_WEBHOOK_URL` or `DISCORD_WEBHOOK_URL` must be set to receive notifications.

### Alertmanager integration

Add a receiver and route to your `alertmanager.yml`:

```yaml
route:
  receiver: default
  routes:
    - match_re:
        severity: warning|critical
      receiver: alertmind
      continue: true   # still sends to your existing receiver

receivers:
  - name: alertmind
    webhook_configs:
      - url: http://alertmind:8080/webhook
        send_resolved: true
```

Set `continue: true` to forward to your existing notification channel alongside alertmind.

### Runbook enrichment

alertmind fetches and includes runbook content when a `runbook_url` annotation is present:

```yaml
# prometheus alert rule
- alert: HighMemoryUsage
  expr: ...
  annotations:
    summary: "Memory usage above 90% on {{ $labels.instance }}"
    runbook_url: "https://your-wiki/runbooks/high-memory"
```

The first 3,000 characters of the runbook are included in the Claude prompt, giving the
model specific context for your environment.

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness check — returns `{"status":"ok"}` |
| `POST` | `/webhook` | Alertmanager webhook receiver |
| `POST` | `/test` | Fire a synthetic alert to validate the pipeline |

Test the full pipeline without a real alert:

```bash
curl -X POST http://localhost:8080/test
```

## Docker Compose

```yaml
services:
  alertmind:
    image: ghcr.io/woodhead-tech/alertmind:latest
    ports:
      - "8080:8080"
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      SLACK_WEBHOOK_URL: ${SLACK_WEBHOOK_URL}
      DISCORD_WEBHOOK_URL: ${DISCORD_WEBHOOK_URL}
    restart: unless-stopped
```

## Design

- **No external Go dependencies** — stdlib only. The Anthropic API is called directly over HTTP.
- **Non-blocking webhook handler** — responds 200 immediately so Alertmanager never times out; triage runs in a background goroutine.
- **Graceful degradation** — if the Claude API call fails, a fallback notification is sent so the alert is never silently dropped.
- **Alert groups** — processes all alerts in a group together in a single LLM call, so correlated alerts are triaged with shared context.

## Cost

alertmind uses `claude-haiku-4-5-20251001` by default — the fastest and cheapest Claude model.
A typical alert triage call uses ~500–800 input tokens and ~300–500 output tokens.
At current Haiku pricing, that's less than $0.001 per alert.

Override with `ALERTMIND_MODEL` to use a more capable model for complex environments.

## License

MIT
