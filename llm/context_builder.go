package llm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yourorg/gorview/core"
)

// BuildPrompt creates a concise LLM prompt for a single finding.
// Raw source code is never included — only the problem summary (name, metrics, location).
func BuildPrompt(f core.Finding) string {
	var sb strings.Builder
	sb.WriteString("You are a software architect expert in Go design patterns.\n")
	sb.WriteString("A static analysis tool detected an architectural smell:\n\n")
	fmt.Fprintf(&sb, "Smell type       : %s\n", f.SmellType)
	fmt.Fprintf(&sb, "Component        : %s\n", f.Component)
	fmt.Fprintf(&sb, "Location         : %s:%d\n", f.File, f.Line)
	if len(f.Metrics) > 0 {
		fmt.Fprintf(&sb, "Metrics          : %s\n", metricsText(f.Metrics))
	}
	fmt.Fprintf(&sb, "Suggested pattern: %s\n\n", f.Pattern)
	sb.WriteString("CRITICAL: Output ONLY a raw JSON object — no markdown, no prose, no code fences.\n")
	sb.WriteString("Your response must start with { and end with }.\n\n")
	sb.WriteString(`{"explanation":"1-2 sentence diagnosis","refactor_before":"Go code ≤15 lines","refactor_after":"Go code ≤15 lines","effort":"Faible|Moyen|Élevé"}`)
	return sb.String()
}

func metricsText(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}
