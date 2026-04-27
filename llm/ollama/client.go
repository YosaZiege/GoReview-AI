package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/llm"
)

const defaultBaseURL = "http://localhost:11434"

// Client calls a local Ollama instance.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

// New creates an Ollama client. If baseURL is empty, defaults to localhost:11434.
func New(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{baseURL: baseURL, model: model, http: &http.Client{}}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

type enrichResult struct {
	Explanation    string `json:"explanation"`
	RefactorBefore string `json:"refactor_before"`
	RefactorAfter  string `json:"refactor_after"`
	Effort         string `json:"effort"`
}

// Enrich sends each finding to Ollama and populates LLM fields. Errors are best-effort.
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
	body, _ := json.Marshal(generateRequest{Model: c.model, Prompt: prompt, Stream: false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: status %d", resp.StatusCode)
	}
	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, err
	}
	var result enrichResult
	if err := json.Unmarshal([]byte(gr.Response), &result); err != nil {
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
