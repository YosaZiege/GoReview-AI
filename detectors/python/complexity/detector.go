package complexity

import (
	"fmt"

	"github.com/yourorg/gorview/core"
	pythonlang "github.com/yourorg/gorview/languages/python"
)

const ccThreshold = 10

// Detector flags Python functions and methods with high cyclomatic complexity.
type Detector struct{}

func (Detector) Name() string { return "high_complexity" }

func (Detector) Detect(files []pythonlang.ParsedFile) []core.Finding {
	var findings []core.Finding
	for _, pf := range files {
		for _, fi := range pf.Funcs {
			if fi.Complexity <= ccThreshold {
				continue
			}
			component := fi.Name
			if fi.Class != "" {
				component = fmt.Sprintf("%s.%s", fi.Class, fi.Name)
			}
			findings = append(findings, core.Finding{
				Severity:  severityFor(fi.Complexity),
				SmellType: "high_complexity",
				File:      pf.Path,
				Line:      fi.Line,
				Component: component,
				Metrics:   map[string]int{"cc": fi.Complexity},
				Pattern:   "Stratégie",
				Effort:    effortFor(fi.Complexity),
			})
		}
	}
	return findings
}

func severityFor(cc int) core.Severity {
	switch {
	case cc > 20:
		return core.SeverityCritical
	case cc > 15:
		return core.SeverityMedium
	default:
		return core.SeverityLow
	}
}

func effortFor(cc int) core.EffortLevel {
	switch {
	case cc > 20:
		return core.EffortHigh
	case cc > 15:
		return core.EffortMedium
	default:
		return core.EffortLow
	}
}
