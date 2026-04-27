package core

// Calculate computes a maintainability score from 0–100.
// Deductions: CRITIQUE = -15, MOYEN = -7, FAIBLE = -3. Floor at 0.
func Calculate(findings []Finding) int {
	score := 100
	for _, f := range findings {
		switch f.Severity {
		case SeverityCritical:
			score -= 15
		case SeverityMedium:
			score -= 7
		case SeverityLow:
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}
