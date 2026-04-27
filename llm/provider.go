package llm

import (
	"context"

	"github.com/yourorg/gorview/core"
)

// Provider enriches findings with LLM-generated explanations and refactoring examples.
type Provider interface {
	Enrich(ctx context.Context, findings []core.Finding) ([]core.Finding, error)
}
