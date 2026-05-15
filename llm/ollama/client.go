package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/llm"
)

const (
	defaultBaseURL = "http://localhost:11434"
	defaultModel   = "qwen2.5-coder:7b"
)

// Client calls a local Ollama instance.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

// New creates an Ollama client. If baseURL is empty, defaults to localhost:11434.
// If model is empty or "llama3" (old default), falls back to qwen2.5-coder:7b.
func New(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if model == "" || model == "llama3" {
		model = defaultModel
	}
	return &Client{baseURL: baseURL, model: model, http: &http.Client{Timeout: 300 * time.Second}}
}

type generateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
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
	if err := c.ensureReady(ctx); err != nil {
		return findings, fmt.Errorf("ollama: %w", err)
	}
	enriched := make([]core.Finding, len(findings))
	copy(enriched, findings)
	for i, f := range enriched {
		result, err := c.call(ctx, llm.BuildPrompt(f))
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorview: ollama enrich %s: %v\n", f.SmellType, err)
			continue
		}
		applyResult(&enriched[i], result)
	}
	return enriched, nil
}

func (c *Client) call(ctx context.Context, prompt string) (*enrichResult, error) {
	raw, err := c.generate(ctx, prompt)
	if err != nil {
		return nil, err
	}
	// Extract JSON object — local LLMs often wrap output in markdown fences or prose.
	raw = extractJSON(raw, '{', '}')
	var result enrichResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}

// extractJSON returns the first substring delimited by open/close from raw.
func extractJSON(raw string, open, close rune) string {
	start := strings.IndexRune(raw, open)
	end := strings.LastIndexAny(raw, string(close))
	if start == -1 || end == -1 || end <= start {
		return raw
	}
	return raw[start : end+1]
}

func applyResult(f *core.Finding, r *enrichResult) {
	f.Explanation = r.Explanation
	f.RefactorBefore = r.RefactorBefore
	f.RefactorAfter = r.RefactorAfter
	if r.Effort != "" {
		f.Effort = core.EffortLevel(r.Effort)
	}
}

func (c *Client) generate(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.1,
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama: status %d: %s", resp.StatusCode, string(errBody))
	}
	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}
	return gr.Response, nil
}

// ensureReady guarantees Ollama is running and the model is pulled.
func (c *Client) ensureReady(ctx context.Context) error {
	if !ollamaResponds(ctx, c.baseURL) {
		fmt.Fprintln(os.Stderr, "[gorview] Starting Ollama...")
		if err := startOllama(); err != nil {
			return err
		}
		// Wait up to 15 seconds for Ollama to come up.
		for i := 0; i < 5; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
			}
			if ollamaResponds(ctx, c.baseURL) {
				break
			}
			if i == 4 {
				return fmt.Errorf("ollama did not respond after 15s — is it installed?")
			}
		}
	}
	return c.ensureModel(ctx)
}

func ollamaResponds(ctx context.Context, baseURL string) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func startOllama() error {
	cmd := exec.Command("ollama", "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func (c *Client) ensureModel(ctx context.Context) error {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return err
	}
	for _, m := range tags.Models {
		if m.Name == c.model || m.Name == c.model+":latest" || strings.HasPrefix(m.Name, c.model) {
			return nil
		}
	}

	fmt.Fprintf(os.Stderr, "[gorview] Pulling model %s (first run only)...\n", c.model)
	cmd := exec.CommandContext(ctx, "ollama", "pull", c.model)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pull model %s: %w", c.model, err)
	}
	return nil
}
