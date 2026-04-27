package core

// Report holds the full analysis result for a directory.
type Report struct {
	Dir      string
	Findings []Finding
	Score    int
}

// NewReport creates a Report from a set of findings, calculating the score.
func NewReport(dir string, findings []Finding) Report {
	return Report{
		Dir:      dir,
		Findings: findings,
		Score:    Calculate(findings),
	}
}

// CountBySeverity returns a map of severity → count.
func (r Report) CountBySeverity() map[Severity]int {
	counts := map[Severity]int{
		SeverityCritical: 0,
		SeverityMedium:   0,
		SeverityLow:      0,
	}
	for _, f := range r.Findings {
		counts[f.Severity]++
	}
	return counts
}
