package godclass

import (
	"github.com/yourorg/gorview/core"
	pythonlang "github.com/yourorg/gorview/languages/python"
)

const (
	methodThreshold = 15
	fieldThreshold  = 10
)

// Detector flags Python classes with too many methods or fields (God Object smell).
type Detector struct{}

func (Detector) Name() string { return "god_class" }

func (Detector) Detect(files []pythonlang.ParsedFile) []core.Finding {
	var findings []core.Finding
	for _, pf := range files {
		for _, ci := range pf.Classes {
			if ci.Methods <= methodThreshold && ci.Fields <= fieldThreshold {
				continue
			}
			findings = append(findings, core.Finding{
				Severity:  severityFor(ci),
				SmellType: "god_class",
				File:      pf.Path,
				Line:      ci.Line,
				Component: ci.Name,
				Metrics:   map[string]int{"methods": ci.Methods, "fields": ci.Fields},
				Pattern:   "Façade",
				Effort:    effortFor(ci),
			})
		}
	}
	return findings
}

func severityFor(ci *pythonlang.ClassInfo) core.Severity {
	if ci.Methods > methodThreshold*2 || ci.Fields > fieldThreshold*2 {
		return core.SeverityCritical
	}
	return core.SeverityMedium
}

func effortFor(ci *pythonlang.ClassInfo) core.EffortLevel {
	if ci.Methods > methodThreshold*2 || ci.Fields > fieldThreshold*2 {
		return core.EffortHigh
	}
	return core.EffortMedium
}
