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

// Analyse sends a per-file ASTSummary to Ollama (legacy path).
func (c *Client) Analyse(ctx context.Context, summary core.ASTSummary, existing []core.Finding) ([]core.Finding, error) {
	raw, err := c.generate(ctx, llm.BuildDeepPrompt(summary, existing))
	if err != nil {
		return nil, err
	}
	return llm.ParseDeepFindings(summary.File, raw), nil
}

// AnalysePackage sends a full package skeleton to Ollama for deep analysis.
// One call covers all files in the package directory, giving cross-file context.
func (c *Client) AnalysePackage(ctx context.Context, sketch core.PackageSketch, existing []core.Finding) ([]core.Finding, error) {
	raw, err := c.generate(ctx, llm.BuildPackagePrompt(sketch, existing))
	if err != nil {
		return nil, err
	}
	return llm.ParseDeepFindings(sketch.Dir, raw), nil
}

func (c *Client) generate(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(generateRequest{Model: c.model, Prompt: prompt, Stream: false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: status %d", resp.StatusCode)
	}
	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", err
	}
	return gr.Response, nil
}
