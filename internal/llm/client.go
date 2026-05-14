package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/whitenhiemer/alertmind/internal/alert"
)

const (
	apiURL    = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	maxTokens  = 1024

	systemPrompt = `You are an on-call triage assistant for engineering teams. Analyze firing alerts and return structured triage information.

Respond with ONLY valid JSON — no markdown, no code blocks, no explanation.

Use this exact structure:
{
  "probable_cause": "most likely root cause in 1-2 sentences",
  "severity_assessment": "urgency and blast radius in 1-2 sentences",
  "immediate_actions": ["ordered list of the first 3-5 things to do right now"],
  "investigation_commands": ["specific shell commands to diagnose the issue"],
  "notes": "any additional context or caveats (optional, can be empty string)"
}

Be direct and specific. Name actual commands, not vague suggestions. If context is insufficient, say so in probable_cause rather than guessing.`
)

type Client struct {
	apiKey string
	model  string
	apiURL string
	http   *http.Client
}

func New(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		apiURL: apiURL,
		http:   &http.Client{},
	}
}

func newWithURL(apiKey, model, url string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		apiURL: url,
		http:   &http.Client{},
	}
}

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) Triage(ctx context.Context, alertContext string) (*alert.Triage, error) {
	reqBody := messagesRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []message{
			{Role: "user", Content: alertContext},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp messagesResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("anthropic API error: %s", apiResp.Error.Message)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	text := extractJSON(apiResp.Content[0].Text)
	var triage alert.Triage
	if err := json.Unmarshal([]byte(text), &triage); err != nil {
		return nil, fmt.Errorf("parse triage JSON: %w\nraw: %s", err, text)
	}
	return &triage, nil
}

// extractJSON strips markdown code fences if the model wraps its output.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return strings.TrimSpace(s)
}
