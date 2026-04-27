package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/llm"
)

const apiURL = "https://api.openai.com/v1/chat/completions"

// Client calls the OpenAI Chat Completions API.
type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

// New creates an OpenAI client.
func New(apiKey, model string) *Client {
	return &Client{apiKey: apiKey, model: model, http: &http.Client{}}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

type enrichResult struct {
	Explanation    string `json:"explanation"`
	RefactorBefore string `json:"refactor_before"`
	RefactorAfter  string `json:"refactor_after"`
	Effort         string `json:"effort"`
}

// Enrich sends each finding to OpenAI and populates LLM fields. Errors are best-effort.
func (c *Client) Enrich(ctx context.Context, findings []core.Finding) ([]core.Finding, error) {
	enriched := make([]core.Finding, len(findings))
	copy(enriched, findings)
	for i, f := range enriched {
		result, err := c.call(ctx, llm.BuildPrompt(f))
		if err != nil {
			continue
		}
		applyResult(&enriched[i], result)
	}
	return enriched, nil
}

func (c *Client) call(ctx context.Context, prompt string) (*enrichResult, error) {
	body, _ := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    []message{{Role: "user", Content: prompt}},
		Temperature: 0.2,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: status %d", resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, err
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}
	var result enrichResult
	if err := json.Unmarshal([]byte(cr.Choices[0].Message.Content), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func applyResult(f *core.Finding, r *enrichResult) {
	f.Explanation = r.Explanation
	f.RefactorBefore = r.RefactorBefore
	f.RefactorAfter = r.RefactorAfter
	if r.Effort != "" {
		f.Effort = core.EffortLevel(r.Effort)
	}
}
