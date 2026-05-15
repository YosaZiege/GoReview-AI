package ollama

import (
	"context"
	"fmt"
	"os"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/llm"
)

// Analyse sends a per-file ASTSummary to Ollama (legacy path).
func (c *Client) Analyse(ctx context.Context, summary core.ASTSummary, existing []core.Finding) ([]core.Finding, error) {
	if err := c.ensureReady(ctx); err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	raw, err := c.generate(ctx, llm.BuildDeepPrompt(summary, existing))
	if err != nil {
		return nil, err
	}
	findings := llm.ParseDeepFindings(summary.File, raw)
	if len(findings) == 0 {
		fmt.Fprintf(os.Stderr, "[gorview] deep: no findings parsed from LLM response for %s\n", summary.File)
	}
	return findings, nil
}

// AnalysePackage sends a full package skeleton to Ollama for deep analysis.
func (c *Client) AnalysePackage(ctx context.Context, sketch core.PackageSketch, existing []core.Finding) ([]core.Finding, error) {
	if err := c.ensureReady(ctx); err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	raw, err := c.generate(ctx, llm.BuildPackagePrompt(sketch, existing))
	if err != nil {
		return nil, err
	}
	findings := llm.ParseDeepFindings(sketch.Dir, raw)
	if len(findings) == 0 {
		fmt.Fprintf(os.Stderr, "[gorview] deep: no findings parsed from LLM response for %s\n", sketch.Dir)
	}
	return findings, nil
}
