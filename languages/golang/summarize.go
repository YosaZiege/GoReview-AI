package golang

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/yourorg/gorview/core"
)

// Summarize converts a ParsedFile (one .go source file) into a token-efficient
// ASTSummary suitable for LLM deep analysis.
func Summarize(pf ParsedFile) core.ASTSummary {
	f := pf.File
	fset := pf.Fset

	sum := core.ASTSummary{File: pf.Path}
	if f.Name != nil {
		sum.Package = f.Name.Name
	}

	for _, imp := range f.Imports {
		path := imp.Path.Value
		if len(path) >= 2 {
			path = path[1 : len(path)-1] // strip quotes
		}
		sum.Imports = append(sum.Imports, path)
	}

	// First pass: collect struct type definitions.
	typeIndex := map[string]int{} // type name → index in sum.Types
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			tn := core.TypeNode{
				Name: ts.Name.Name,
				Kind: "struct",
				Line: fset.Position(ts.Pos()).Line,
			}
			if st.Fields != nil {
				for _, field := range st.Fields.List {
					typName := exprStr(field.Type)
					if len(field.Names) == 0 {
						tn.Fields = append(tn.Fields, core.FieldNode{Name: typName, Type: typName})
					} else {
						for _, name := range field.Names {
							tn.Fields = append(tn.Fields, core.FieldNode{Name: name.Name, Type: typName})
						}
					}
				}
			}
			typeIndex[tn.Name] = len(sum.Types)
			sum.Types = append(sum.Types, tn)
		}
	}

	// Second pass: collect functions and methods.
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name == nil {
			continue
		}
		mn := buildMethodNode(fd, fset)

		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			recvName := receiverTypeName(fd.Recv.List[0].Type)
			if idx, exists := typeIndex[recvName]; exists {
				sum.Types[idx].Methods = append(sum.Types[idx].Methods, mn)
			} else {
				// Type is defined in another file of the same package; create a stub.
				tn := core.TypeNode{Name: recvName, Kind: "struct"}
				tn.Methods = append(tn.Methods, mn)
				typeIndex[recvName] = len(sum.Types)
				sum.Types = append(sum.Types, tn)
			}
		} else {
			sum.Funcs = append(sum.Funcs, core.FuncNode{
				Name:       mn.Name,
				Line:       mn.Line,
				Params:     mn.Params,
				Complexity: mn.Complexity,
				Lines:      mn.Lines,
				Calls:      mn.Calls,
			})
		}
	}

	return sum
}

func buildMethodNode(fd *ast.FuncDecl, fset *token.FileSet) core.MethodNode {
	return core.MethodNode{
		Name:       fd.Name.Name,
		Line:       fset.Position(fd.Pos()).Line,
		Params:     extractParams(fd.Type.Params),
		Complexity: computeCC(fd),
		Lines:      fset.Position(fd.End()).Line - fset.Position(fd.Pos()).Line + 1,
		Calls:      extractCalls(fd.Body),
	}
}

func computeCC(fd *ast.FuncDecl) int {
	cc := 1
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.IfStmt:
			cc++
		case *ast.ForStmt:
			cc++
		case *ast.RangeStmt:
			cc++
		case *ast.CaseClause:
			if v.List != nil {
				cc++
			}
		case *ast.CommClause:
			if v.Comm != nil {
				cc++
			}
		case *ast.BinaryExpr:
			if v.Op == token.LAND || v.Op == token.LOR {
				cc++
			}
		}
		return true
	})
	return cc
}

// extractCalls collects unique selector-expression call targets (e.g. "db.Query")
// from a function body, capped at 10 to keep the summary token-efficient.
func extractCalls(body *ast.BlockStmt) []string {
	if body == nil {
		return nil
	}
	seen := map[string]bool{}
	var calls []string
	ast.Inspect(body, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		call := id.Name + "." + sel.Sel.Name
		if !seen[call] && len(calls) < 10 {
			seen[call] = true
			calls = append(calls, call)
		}
		return true
	})
	return calls
}

func extractParams(fl *ast.FieldList) []string {
	if fl == nil {
		return nil
	}
	var params []string
	for _, field := range fl.List {
		t := exprStr(field.Type)
		if len(field.Names) == 0 {
			params = append(params, t)
		} else {
			for _, name := range field.Names {
				params = append(params, fmt.Sprintf("%s %s", name.Name, t))
			}
		}
	}
	return params
}

func exprStr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprStr(t.X)
	case *ast.SelectorExpr:
		return exprStr(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprStr(t.Elt)
	case *ast.MapType:
		return "map[" + exprStr(t.Key) + "]" + exprStr(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chan " + exprStr(t.Value)
	case *ast.Ellipsis:
		return "..." + exprStr(t.Elt)
	case *ast.FuncType:
		return "func(...)"
	default:
		return "?"
	}
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}
