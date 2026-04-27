package core

type Severity string

const (
	SeverityCritical Severity = "CRITIQUE"
	SeverityMedium   Severity = "MOYEN"
	SeverityLow      Severity = "FAIBLE"
)

type EffortLevel string

const (
	EffortHigh   EffortLevel = "Élevé"
	EffortMedium EffortLevel = "Moyen"
	EffortLow    EffortLevel = "Faible"
)

type Finding struct {
	Severity    Severity
	SmellType   string         // "god_struct", "concrete_dep", "high_complexity", etc.
	File        string
	Line        int
	Component   string         // name of the struct/function
	Metrics     map[string]int // e.g. {"fields": 19, "methods": 28}
	Pattern     string         // suggested design pattern name
	Effort      EffortLevel
	// Populated by LLM enrichment only:
	Explanation    string
	RefactorBefore string
	RefactorAfter  string
}
