package output

import (
	"encoding/json"
	"io"

	"github.com/yourorg/gorview/core"
)

type jsonFinding struct {
	Severity       string         `json:"severity"`
	SmellType      string         `json:"smell_type"`
	File           string         `json:"file"`
	Line           int            `json:"line"`
	Component      string         `json:"component"`
	Metrics        map[string]int `json:"metrics"`
	Pattern        string         `json:"pattern"`
	Effort         string         `json:"effort"`
	Explanation    string         `json:"explanation,omitempty"`
	RefactorBefore string         `json:"refactor_before,omitempty"`
	RefactorAfter  string         `json:"refactor_after,omitempty"`
}

type jsonReport struct {
	Dir      string        `json:"dir"`
	Score    int           `json:"score"`
	Findings []jsonFinding `json:"findings"`
}

// PrintJSON writes the report as indented JSON to w.
func PrintJSON(w io.Writer, r core.Report) error {
	jr := jsonReport{Dir: r.Dir, Score: r.Score}
	for _, f := range r.Findings {
		jr.Findings = append(jr.Findings, jsonFinding{
			Severity:       string(f.Severity),
			SmellType:      f.SmellType,
			File:           f.File,
			Line:           f.Line,
			Component:      f.Component,
			Metrics:        f.Metrics,
			Pattern:        f.Pattern,
			Effort:         string(f.Effort),
			Explanation:    f.Explanation,
			RefactorBefore: f.RefactorBefore,
			RefactorAfter:  f.RefactorAfter,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}
