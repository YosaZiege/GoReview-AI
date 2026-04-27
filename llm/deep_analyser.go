package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/yourorg/gorview/core"
)

// DeepAnalyser performs semantic analysis on source skeletons and returns
// findings that complement the static detector pass.
type DeepAnalyser interface {
	// Analyse performs per-file analysis using a JSON AST summary (legacy path).
	Analyse(ctx context.Context, summary core.ASTSummary, existing []core.Finding) ([]core.Finding, error)
	// AnalysePackage performs package-level analysis using stripped source code.
	// This is the preferred path: the LLM sees real syntax across all files in
	// a package directory, giving it cross-file context at low token cost.
	AnalysePackage(ctx context.Context, sketch core.PackageSketch, existing []core.Finding) ([]core.Finding, error)
}

type deepFinding struct {
	SmellType      string `json:"smell_type"`
	Component      string `json:"component"`
	File           string `json:"file"`           // optional; specific file within the package
	Line           int    `json:"line"`
	Severity       string `json:"severity"`
	Explanation    string `json:"explanation"`
	Pattern        string `json:"pattern"`
	Effort         string `json:"effort"`
	RefactorBefore string `json:"refactor_before"`
	RefactorAfter  string `json:"refactor_after"`
}

// ParseDeepFindings extracts core.Finding values from a raw LLM JSON response.
// defaultFile is used when the LLM does not specify a file in the finding.
func ParseDeepFindings(defaultFile, raw string) []core.Finding {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	raw = raw[start : end+1]

	var df []deepFinding
	if err := json.Unmarshal([]byte(raw), &df); err != nil {
		return nil
	}

	var findings []core.Finding
	for _, d := range df {
		file := defaultFile
		if d.File != "" {
			file = d.File
		}
		f := core.Finding{
			SmellType:      d.SmellType,
			File:           file,
			Line:           d.Line,
			Component:      d.Component,
			Pattern:        d.Pattern,
			Explanation:    d.Explanation,
			RefactorBefore: d.RefactorBefore,
			RefactorAfter:  d.RefactorAfter,
		}
		switch strings.ToUpper(d.Severity) {
		case "CRITIQUE":
			f.Severity = core.SeverityCritical
		case "MOYEN":
			f.Severity = core.SeverityMedium
		default:
			f.Severity = core.SeverityLow
		}
		switch d.Effort {
		case "Élevé":
			f.Effort = core.EffortHigh
		case "Moyen":
			f.Effort = core.EffortMedium
		default:
			f.Effort = core.EffortLow
		}
		findings = append(findings, f)
	}
	return findings
}
