package llm

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yourorg/gorview/core"
)

// BuildDeepPrompt creates a per-file prompt from a JSON ASTSummary (legacy path).
func BuildDeepPrompt(summary core.ASTSummary, existing []core.Finding) string {
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")

	var skip []string
	for _, f := range existing {
		if f.File == summary.File {
			skip = append(skip, fmt.Sprintf("%s at line %d (%s)", f.SmellType, f.Line, f.Component))
		}
	}

	var sb strings.Builder
	sb.WriteString("You are an expert software architect performing deep architectural analysis.\n")
	sb.WriteString("Below is a structured AST summary of a source file — NOT raw source code.\n\n")

	if len(skip) > 0 {
		sb.WriteString("Already detected by static analysis — do NOT re-flag these:\n")
		for _, s := range skip {
			fmt.Fprintf(&sb, "  - %s\n", s)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Identify architectural smells not listed above.\n")
	sb.WriteString("Focus on: layer violations, feature envy, missing abstractions, temporal coupling, anemic domain model.\n\n")
	sb.WriteString("Respond ONLY with a JSON array ([] if nothing found). Each element:\n")
	sb.WriteString(`{"smell_type":"snake_case","component":"Type.method or func","line":0,`)
	sb.WriteString(`"severity":"CRITIQUE|MOYEN|FAIBLE","explanation":"one sentence",`)
	sb.WriteString(`"pattern":"design pattern","effort":"Faible|Moyen|Élevé",`)
	sb.WriteString(`"refactor_before":"snippet ≤10 lines","refactor_after":"snippet ≤10 lines"}`)
	sb.WriteString("\n\nAST summary:\n")
	sb.Write(summaryJSON)

	return sb.String()
}

// BuildPackagePrompt creates a package-level prompt from stripped source code.
// The LLM sees real language syntax across all files in the package, enabling
// cross-file pattern detection at low token cost.
func BuildPackagePrompt(sketch core.PackageSketch, existing []core.Finding) string {
	// Only list findings relevant to files in this package.
	fileSet := map[string]bool{}
	for _, p := range sketch.Files {
		fileSet[p] = true
	}
	var skip []string
	for _, f := range existing {
		if fileSet[f.File] {
			skip = append(skip, fmt.Sprintf(
				"%s at line %d in %s (%s)",
				f.SmellType, f.Line, filepath.Base(f.File), f.Component,
			))
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "You are an expert software architect performing deep architectural analysis.\n")
	fmt.Fprintf(&sb, "Below is the skeleton of a %s package at: %s\n", sketch.Language, sketch.Dir)
	sb.WriteString("Function bodies have been stripped — only signatures, types, and structure remain.\n\n")

	if len(skip) > 0 {
		sb.WriteString("Already detected by static analysis — do NOT re-flag:\n")
		for _, s := range skip {
			fmt.Fprintf(&sb, "  - %s\n", s)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Identify architectural smells across ALL files in this package.\n")
	sb.WriteString("Cross-file patterns are especially valuable: feature envy, layer violations, missing abstractions.\n")
	sb.WriteString("For each finding, include the exact file path from the === FILE headers.\n\n")
	sb.WriteString("Return ONLY a JSON array ([] if nothing found). Each element:\n")
	sb.WriteString(`{"smell_type":"snake_case","component":"Type.method or func",`)
	sb.WriteString(`"file":"exact/path/from/header.go","line":0,`)
	sb.WriteString(`"severity":"CRITIQUE|MOYEN|FAIBLE","explanation":"one sentence",`)
	sb.WriteString(`"pattern":"design pattern","effort":"Faible|Moyen|Élevé",`)
	sb.WriteString(`"refactor_before":"snippet ≤10 lines","refactor_after":"snippet ≤10 lines"}`)
	sb.WriteString("\n\nPackage skeleton:\n```\n")
	sb.WriteString(sketch.Source)
	sb.WriteString("```\n")

	return sb.String()
}
