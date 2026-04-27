package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/yourorg/gorview/core"
)

// PrintTerminal writes a human-readable, colored report to w.
func PrintTerminal(w io.Writer, r core.Report) {
	findings := make([]core.Finding, len(r.Findings))
	copy(findings, r.Findings)
	sort.Slice(findings, func(i, j int) bool {
		return severityOrder(findings[i].Severity) < severityOrder(findings[j].Severity)
	})

	for _, f := range findings {
		printFinding(w, f)
	}
	printScore(w, r)
}

func printFinding(w io.Writer, f core.Finding) {
	sevLabel := severityColor(f.Severity).Sprintf("[%s]", string(f.Severity))
	loc := fmt.Sprintf("%s:%d", f.File, f.Line)
	fmt.Fprintf(w, "%-22s %-20s %s\n", sevLabel, f.SmellType, loc)
	fmt.Fprintf(w, "             %s\n", describeComponent(f))
	fmt.Fprintf(w, "             → Patron suggéré : %s · Effort : %s\n\n",
		color.CyanString(f.Pattern),
		effortColor(f.Effort).Sprint(string(f.Effort)),
	)
	if f.Explanation != "" {
		fmt.Fprintf(w, "             %s\n\n", color.HiBlackString(f.Explanation))
	}
	if f.RefactorBefore != "" && f.RefactorAfter != "" {
		fmt.Fprintf(w, "             Before:\n%s\n\n             After:\n%s\n\n",
			indent(f.RefactorBefore), indent(f.RefactorAfter))
	}
}

func printScore(w io.Writer, r core.Report) {
	counts := r.CountBySeverity()
	fmt.Fprintf(w, "score de maintenabilité : %s\n",
		scoreColorFor(r.Score).Sprintf("%d / 100", r.Score))
	fmt.Fprintf(w, "  critique : %d  moyen : %d  faible : %d\n",
		counts[core.SeverityCritical],
		counts[core.SeverityMedium],
		counts[core.SeverityLow],
	)
}

func describeComponent(f core.Finding) string {
	switch f.SmellType {
	case "god_struct":
		parts := []string{f.Component}
		if v, ok := f.Metrics["fields"]; ok {
			parts = append(parts, fmt.Sprintf("%d champs", v))
		}
		if v, ok := f.Metrics["methods"]; ok {
			parts = append(parts, fmt.Sprintf("%d méthodes", v))
		}
		return strings.Join(parts, " · ")
	case "concrete_dep":
		return f.Component + " est un type concret, pas une interface"
	case "high_complexity":
		if cc, ok := f.Metrics["cc"]; ok {
			return fmt.Sprintf("%s · CC=%d", f.Component, cc)
		}
		return f.Component
	default:
		return f.Component
	}
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "               " + l
	}
	return strings.Join(lines, "\n")
}

func severityColor(s core.Severity) *color.Color {
	switch s {
	case core.SeverityCritical:
		return color.New(color.FgRed, color.Bold)
	case core.SeverityMedium:
		return color.New(color.FgYellow, color.Bold)
	default:
		return color.New(color.FgBlue)
	}
}

func effortColor(e core.EffortLevel) *color.Color {
	switch e {
	case core.EffortHigh:
		return color.New(color.FgRed)
	case core.EffortMedium:
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgGreen)
	}
}

func scoreColorFor(score int) *color.Color {
	switch {
	case score >= 80:
		return color.New(color.FgGreen, color.Bold)
	case score >= 60:
		return color.New(color.FgYellow, color.Bold)
	default:
		return color.New(color.FgRed, color.Bold)
	}
}

func severityOrder(s core.Severity) int {
	switch s {
	case core.SeverityCritical:
		return 0
	case core.SeverityMedium:
		return 1
	default:
		return 2
	}
}
