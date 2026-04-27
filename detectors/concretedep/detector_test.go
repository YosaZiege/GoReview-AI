package concretedep_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/detectors/concretedep"
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

func TestConcreteDep(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantN     int
		wantComp  string // expected Component if wantN == 1
		wantSev   core.Severity
	}{
		{
			name: "injectable field — pointer to named type",
			src: `package p
type OrderHandler struct {
	db *PostgresDB
}`,
			wantN:    1,
			wantComp: "OrderHandler.db",
			wantSev:  core.SeverityMedium,
		},
		{
			name: "svc field — pointer to concrete service",
			src: `package p
type API struct {
	svc *UserService
}`,
			wantN:    1,
			wantComp: "API.svc",
			wantSev:  core.SeverityMedium,
		},
		{
			name: "non-injectable field name — no smell",
			src: `package p
type Counter struct {
	count *BigInt
}`,
			wantN: 0,
		},
		{
			name: "injectable name but non-pointer type — no smell (not pointer to named)",
			src: `package p
type Repo struct {
	db int
}`,
			wantN: 0,
		},
		{
			name: "multiple injectable fields",
			src: `package p
type Service struct {
	db    *PostgresDB
	cache *RedisCache
	logger *ZapLogger
}`,
			wantN: 3,
		},
		{
			name: "embedded struct — ignored (no names)",
			src: `package p
type Foo struct {
	*Base
}`,
			wantN: 0,
		},
		{
			name: "plain struct, no smells",
			src: `package p
type Point struct {
	X, Y float64
}`,
			wantN: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := concretedep.Detector{}.Detect([]golang.ParsedFile{mustParse(t, tc.src)})
			if len(findings) != tc.wantN {
				t.Fatalf("want %d findings, got %d: %+v", tc.wantN, len(findings), findings)
			}
			if tc.wantN == 1 {
				if findings[0].Component != tc.wantComp {
					t.Errorf("want component %q, got %q", tc.wantComp, findings[0].Component)
				}
				if findings[0].Severity != tc.wantSev {
					t.Errorf("want severity %s, got %s", tc.wantSev, findings[0].Severity)
				}
				if findings[0].Pattern != "Injection de dépendances" {
					t.Errorf("want pattern 'Injection de dépendances', got %q", findings[0].Pattern)
				}
				if findings[0].Effort != core.EffortLow {
					t.Errorf("want effort Faible, got %s", findings[0].Effort)
				}
			}
		})
	}
}
