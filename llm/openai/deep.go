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

// Analyse sends a per-file ASTSummary to OpenAI (legacy path).
func (c *Client) Analyse(ctx context.Context, summary core.ASTSummary, existing []core.Finding) ([]core.Finding, error) {
	raw, err := c.chat(ctx, llm.BuildDeepPrompt(summary, existing))
	if err != nil {
		return nil, err
	}
	return llm.ParseDeepFindings(summary.File, raw), nil
}

// AnalysePackage sends a full package skeleton to OpenAI for deep analysis.
// One call covers all files in the package directory, giving cross-file context.
func (c *Client) AnalysePackage(ctx context.Context, sketch core.PackageSketch, existing []core.Finding) ([]core.Finding, error) {
	raw, err := c.chat(ctx, llm.BuildPackagePrompt(sketch, existing))
	if err != nil {
		return nil, err
	}
	return llm.ParseDeepFindings(sketch.Dir, raw), nil
}

func (c *Client) chat(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    []message{{Role: "user", Content: prompt}},
		Temperature: 0.2,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai: status %d", resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", err
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices returned")
	}
	return cr.Choices[0].Message.Content, nil
}
