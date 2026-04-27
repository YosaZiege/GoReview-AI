package concretedep

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/languages/golang"
)

// injectableNames are field names that conventionally hold injected dependencies.
var injectableNames = map[string]bool{
	"db": true, "database": true, "store": true, "repo": true, "repository": true,
	"svc": true, "service": true, "client": true, "handler": true, "manager": true,
	"cache": true, "queue": true, "bus": true, "publisher": true, "logger": true,
}

// Detector flags struct fields that hold concrete types where an interface is expected.
type Detector struct{}

func (Detector) Name() string { return "concrete_dep" }

func (Detector) Detect(files []golang.ParsedFile) []core.Finding {
	var findings []core.Finding
	for _, pf := range files {
		findings = append(findings, detectFile(pf)...)
	}
	return findings
}

func detectFile(pf golang.ParsedFile) []core.Finding {
	var findings []core.Finding
	ast.Inspect(pf.File, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		structName := ts.Name.Name
		for _, field := range st.Fields.List {
			if !isConcreteDep(pf, field) {
				continue
			}
			pos := pf.Fset.Position(field.Pos())
			findings = append(findings, core.Finding{
				Severity:  core.SeverityMedium,
				SmellType: "concrete_dep",
				File:      pf.Path,
				Line:      pos.Line,
				Component: structName + "." + fieldDisplayName(field),
				Metrics:   map[string]int{},
				Pattern:   "Injection de dépendances",
				Effort:    core.EffortLow,
			})
		}
		return true
	})
	return findings
}

func isConcreteDep(pf golang.ParsedFile, field *ast.Field) bool {
	// Only check explicitly named fields
	if len(field.Names) == 0 {
		return false
	}
	nameIsInjectable := false
	for _, n := range field.Names {
		if injectableNames[strings.ToLower(n.Name)] {
			nameIsInjectable = true
			break
		}
	}
	if !nameIsInjectable {
		return false
	}

	// With type info: confirm the type is a concrete struct, not an interface
	if pf.Info != nil {
		t := pf.Info.TypeOf(field.Type)
		if t == nil {
			return false
		}
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		named, ok := t.(*types.Named)
		if !ok {
			return false
		}
		_, isStruct := named.Underlying().(*types.Struct)
		return isStruct
	}

	// AST fallback: pointer to a named type is likely concrete
	return isPointerToNamed(field.Type)
}

func isPointerToNamed(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	_, ok = star.X.(*ast.Ident)
	return ok
}

func fieldDisplayName(field *ast.Field) string {
	if len(field.Names) > 0 {
		return field.Names[0].Name
	}
	return typeExprName(field.Type)
}

func typeExprName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeExprName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return "?"
}
