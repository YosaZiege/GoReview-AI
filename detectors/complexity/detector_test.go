package complexity_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/detectors/complexity"
	"github.com/yourorg/gorview/languages/golang"
)

func mustParse(t *testing.T, src string) golang.ParsedFile {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return golang.ParsedFile{Path: "test.go", Fset: fset, File: f}
}

func TestComplexity(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantN   int
		wantCC  int
		wantSev core.Severity
	}{
		{
			name: "simple function — no smell",
			src: `package p
func Simple(x int) int { return x + 1 }`,
			wantN: 0,
		},
		{
			name: "exactly 10 branches — boundary, no smell",
			src: `package p
func Border(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
	if x > 9 { }
}`,
			// CC = 1 + 10 = 11 → wait, 10 ifs gives CC=11, which IS above threshold
			// Let me recalculate: threshold is CC > 10, so 11 > 10 → flagged
			wantN:   1,
			wantCC:  11,
			wantSev: core.SeverityLow,
		},
		{
			name: "9 branches — no smell (CC=10, not above threshold)",
			src: `package p
func JustUnder(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
}`,
			wantN: 0,
		},
		{
			name: "11 ifs — FAIBLE (CC=12, 10 < 12 <= 15)",
			src: `package p
func ManyIfs(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
	if x > 9 { }
	if x > 10 { }
}`,
			wantN:   1,
			wantCC:  12,
			wantSev: core.SeverityLow,
		},
		{
			name: "16 ifs — MOYEN (CC=17, 15 < 17 <= 20)",
			src: `package p
func LotsOfIfs(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
	if x > 9 { }
	if x > 10 { }
	if x > 11 { }
	if x > 12 { }
	if x > 13 { }
	if x > 14 { }
	if x > 15 { }
}`,
			wantN:   1,
			wantCC:  17,
			wantSev: core.SeverityMedium,
		},
		{
			name: "21 ifs — CRITIQUE (CC=22, > 20)",
			src: `package p
func TooComplex(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
	if x > 9 { }
	if x > 10 { }
	if x > 11 { }
	if x > 12 { }
	if x > 13 { }
	if x > 14 { }
	if x > 15 { }
	if x > 16 { }
	if x > 17 { }
	if x > 18 { }
	if x > 19 { }
	if x > 20 { }
}`,
			wantN:   1,
			wantCC:  22,
			wantSev: core.SeverityCritical,
		},
		{
			name: "logical operators add complexity",
			src: `package p
func WithLogic(a, b, c, d, e, f, g, h, i, j, k bool) bool {
	return a && b && c && d && e && f && g && h && i && j && k
}`,
			// CC = 1 + 10 (&&) = 11, > 10 threshold → flagged
			wantN:   1,
			wantCC:  11,
			wantSev: core.SeverityLow,
		},
		{
			name: "for and range add complexity",
			src: `package p
func WithLoops(xs []int) {
	for i := 0; i < 10; i++ {
		if i > 5 { }
		if i > 6 { }
		if i > 7 { }
		if i > 8 { }
	}
	for range xs {
		if len(xs) > 0 { }
		if len(xs) > 1 { }
		if len(xs) > 2 { }
		if len(xs) > 3 { }
	}
}`,
			// CC = 1 + 1(for) + 4(ifs) + 1(range) + 4(ifs) = 11
			wantN:   1,
			wantCC:  11,
			wantSev: core.SeverityLow,
		},
		{
			name: "switch cases add complexity",
			src: `package p
func WithSwitch(x int) {
	switch x {
	case 1: if x > 0 {}
	case 2: if x > 0 {}
	case 3: if x > 0 {}
	case 4: if x > 0 {}
	case 5: if x > 0 {}
	case 6: if x > 0 {}
	case 7: if x > 0 {}
	case 8: if x > 0 {}
	case 9: if x > 0 {}
	case 10: if x > 0 {}
	}
}`,
			// CC = 1 + 10(cases) + 10(ifs) = 21 → CRITIQUE
			wantN:   1,
			wantCC:  21,
			wantSev: core.SeverityCritical,
		},
		{
			name: "method receiver — component includes type name",
			src: `package p
type Parser struct{}
func (p *Parser) Parse(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
	if x > 9 { }
	if x > 10 { }
}`,
			wantN:   1,
			wantCC:  12,
			wantSev: core.SeverityLow,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := complexity.Detector{}.Detect([]golang.ParsedFile{mustParse(t, tc.src)})
			if len(findings) != tc.wantN {
				t.Fatalf("want %d findings, got %d: %+v", tc.wantN, len(findings), findings)
			}
			if tc.wantN > 0 {
				if findings[0].Metrics["cc"] != tc.wantCC {
					t.Errorf("want CC=%d, got %d", tc.wantCC, findings[0].Metrics["cc"])
				}
				if findings[0].Severity != tc.wantSev {
					t.Errorf("want severity %s, got %s", tc.wantSev, findings[0].Severity)
				}
				if findings[0].Pattern != "Stratégie" {
					t.Errorf("want pattern Stratégie, got %s", findings[0].Pattern)
				}
			}
		})
	}
}

func TestComplexity_MethodName(t *testing.T) {
	src := `package p
type Svc struct{}
func (s *Svc) Process(x int) {
	if x > 0 { }
	if x > 1 { }
	if x > 2 { }
	if x > 3 { }
	if x > 4 { }
	if x > 5 { }
	if x > 6 { }
	if x > 7 { }
	if x > 8 { }
	if x > 9 { }
	if x > 10 { }
}`
	findings := complexity.Detector{}.Detect([]golang.ParsedFile{mustParse(t, src)})
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Component != "Svc.Process" {
		t.Errorf("want component Svc.Process, got %s", findings[0].Component)
	}
}
