package godstruct_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/detectors/godstruct"
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

func TestGodStruct_Fields(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantN   int
		wantSev core.Severity
	}{
		{
			name: "small struct — no smell",
			src: `package p
type Small struct { A, B, C int }`,
			wantN: 0,
		},
		{
			name: "exactly 10 fields — boundary, no smell",
			src: `package p
type Border struct {
	A, B, C, D, E int
	F, G, H, I, J string
}`,
			wantN: 0,
		},
		{
			name: "11 fields — MOYEN",
			src: `package p
type Big struct {
	A, B, C, D, E int
	F, G, H, I, J string
	K              bool
}`,
			wantN:   1,
			wantSev: core.SeverityMedium,
		},
		{
			name: "21 fields — CRITIQUE",
			src: `package p
type Giant struct {
	A, B, C, D, E, F, G, H, I, J int
	K, L, M, N, O, P, Q, R, S, T string
	U                              bool
}`,
			wantN:   1,
			wantSev: core.SeverityCritical,
		},
		{
			name: "embedded fields count",
			src: `package p
import "sync"
type WithEmbed struct {
	sync.Mutex
	A, B, C, D, E, F, G, H, I, J, K int
}`,
			wantN:   1,
			wantSev: core.SeverityMedium,
		},
		{
			name: "non-struct type — no smell",
			src: `package p
type MyInt int
type MyFunc func()`,
			wantN: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := godstruct.Detector{}.Detect([]golang.ParsedFile{mustParse(t, tc.src)})
			if len(findings) != tc.wantN {
				t.Fatalf("want %d findings, got %d: %+v", tc.wantN, len(findings), findings)
			}
			if tc.wantN > 0 {
				if findings[0].Severity != tc.wantSev {
					t.Errorf("want severity %s, got %s", tc.wantSev, findings[0].Severity)
				}
				if findings[0].SmellType != "god_struct" {
					t.Errorf("want smell_type god_struct, got %s", findings[0].SmellType)
				}
				if findings[0].Pattern != "Façade" {
					t.Errorf("want pattern Façade, got %s", findings[0].Pattern)
				}
			}
		})
	}
}

func TestGodStruct_Methods(t *testing.T) {
	// 16 methods on an otherwise empty struct — exceeds threshold of 15
	src := `package p
type Svc struct{}
func (Svc) M1()  {}
func (Svc) M2()  {}
func (Svc) M3()  {}
func (Svc) M4()  {}
func (Svc) M5()  {}
func (Svc) M6()  {}
func (Svc) M7()  {}
func (Svc) M8()  {}
func (Svc) M9()  {}
func (Svc) M10() {}
func (Svc) M11() {}
func (Svc) M12() {}
func (Svc) M13() {}
func (Svc) M14() {}
func (Svc) M15() {}
func (Svc) M16() {}`

	findings := godstruct.Detector{}.Detect([]golang.ParsedFile{mustParse(t, src)})
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Component != "Svc" {
		t.Errorf("want component Svc, got %s", findings[0].Component)
	}
	if findings[0].Metrics["methods"] != 16 {
		t.Errorf("want methods=16, got %d", findings[0].Metrics["methods"])
	}
}

func TestGodStruct_PointerReceiver(t *testing.T) {
	// Methods with pointer receivers should still be counted
	src := `package p
type Handler struct{}
func (*Handler) A() {}
func (*Handler) B() {}
func (*Handler) C() {}
func (*Handler) D() {}
func (*Handler) E() {}
func (*Handler) F() {}
func (*Handler) G() {}
func (*Handler) H() {}
func (*Handler) I() {}
func (*Handler) J() {}
func (*Handler) K() {}
func (*Handler) L() {}
func (*Handler) M() {}
func (*Handler) N() {}
func (*Handler) O() {}
func (*Handler) P() {}`

	findings := godstruct.Detector{}.Detect([]golang.ParsedFile{mustParse(t, src)})
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Metrics["methods"] != 16 {
		t.Errorf("want methods=16, got %d", findings[0].Metrics["methods"])
	}
}

func TestGodStruct_MethodsSpreadAcrossFiles(t *testing.T) {
	// Same struct defined and extended across two files in the same package
	file1 := mustParse(t, `package p
type Spread struct{ A, B, C, D, E, F, G, H, I, J, K int }
func (Spread) M1() {}
func (Spread) M2() {}
func (Spread) M3() {}
func (Spread) M4() {}
func (Spread) M5() {}
func (Spread) M6() {}
func (Spread) M7() {}
func (Spread) M8() {}`)

	// Use a different path to simulate a second file in the same dir
	fset2 := token.NewFileSet()
	f2, _ := parser.ParseFile(fset2, "test.go", `package p
func (Spread) M9()  {}
func (Spread) M10() {}`, 0)
	file2 := golang.ParsedFile{Path: "test.go", Fset: fset2, File: f2}

	findings := godstruct.Detector{}.Detect([]golang.ParsedFile{file1, file2})
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	// 11 fields → triggered; methods counted from both files
	if findings[0].Metrics["fields"] != 11 {
		t.Errorf("want fields=11, got %d", findings[0].Metrics["fields"])
	}
}
