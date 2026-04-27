// Package python provides a line-scanner-based structural parser for Python source files.
// It does not perform full type resolution; it extracts class/function structure and
// approximates cyclomatic complexity — enough for architectural smell detection.
package python

import (
	"os"
	"regexp"
	"strings"
)

var (
	reClass  = regexp.MustCompile(`^class\s+(\w+)`)
	reMethod = regexp.MustCompile(`^def\s+\w+\s*\(\s*(?:self|cls)\b`)
	reFunc   = regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	reSelf   = regexp.MustCompile(`^\s+self\.(\w+)\s*=`)
	// keywords that increase cyclomatic complexity (McCabe)
	reCC = regexp.MustCompile(`\b(?:if|elif|for|while|except|and|or)\b`)
)

// ClassInfo holds structural information about a Python class.
type ClassInfo struct {
	Name    string
	Line    int
	Methods int
	Fields  int // approximated from self.x = … in __init__
}

// FuncInfo holds structural information about a Python function or method.
type FuncInfo struct {
	Name       string
	Line       int
	Class      string // empty for module-level functions
	Complexity int    // McCabe CC (starts at 1)
}

// ParsedFile is the result of parsing a single Python source file.
type ParsedFile struct {
	Path    string
	Classes []*ClassInfo
	Funcs   []*FuncInfo
}

type scope struct {
	kind   string // "class" or "func"
	name   string
	indent int
}

// ParseFile parses a single Python source file.
func ParseFile(path string) (ParsedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParsedFile{}, err
	}
	pf := ParsedFile{Path: path}
	classMap := map[string]*ClassInfo{}

	var stack []scope
	var curFunc *FuncInfo

	lines := strings.Split(string(data), "\n")
	for i, raw := range lines {
		lineNum := i + 1
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		indent := indentOf(raw)

		// Pop scopes we have outdented out of.
		for len(stack) > 0 && indent <= stack[len(stack)-1].indent {
			top := stack[len(stack)-1]
			if top.kind == "func" && curFunc != nil {
				pf.Funcs = append(pf.Funcs, curFunc)
				curFunc = nil
			}
			stack = stack[:len(stack)-1]
		}

		// Class definition: class Foo: or class Foo(Base):
		if m := reClass.FindStringSubmatch(stripped); m != nil {
			ci := &ClassInfo{Name: m[1], Line: lineNum}
			pf.Classes = append(pf.Classes, ci)
			classMap[m[1]] = ci
			stack = append(stack, scope{kind: "class", name: m[1], indent: indent})
			continue
		}

		// Function or method definition
		if m := reFunc.FindStringSubmatch(stripped); m != nil {
			if curFunc != nil {
				pf.Funcs = append(pf.Funcs, curFunc)
			}
			funcName := m[1]

			// Determine enclosing class (innermost class scope)
			class := enclosingClass(stack)
			if class != "" && reMethod.MatchString(stripped) {
				classMap[class].Methods++
			} else {
				class = "" // standalone function, not a method
			}

			curFunc = &FuncInfo{Name: funcName, Line: lineNum, Class: class, Complexity: 1}
			stack = append(stack, scope{kind: "func", name: funcName, indent: indent})
			continue
		}

		// Inside a function: accumulate complexity and field assignments
		if curFunc != nil {
			curFunc.Complexity += len(reCC.FindAllString(stripped, -1))

			if curFunc.Name == "__init__" && curFunc.Class != "" {
				if reSelf.MatchString(raw) {
					classMap[curFunc.Class].Fields++
				}
			}
		}
	}

	if curFunc != nil {
		pf.Funcs = append(pf.Funcs, curFunc)
	}
	return pf, nil
}

// ParseDir walks dir and parses every .py file found.
func ParseDir(dir string) ([]ParsedFile, error) {
	paths, err := WalkPyFiles(dir)
	if err != nil {
		return nil, err
	}
	var results []ParsedFile
	for _, p := range paths {
		pf, err := ParseFile(p)
		if err != nil {
			continue // best-effort: skip unreadable files
		}
		results = append(results, pf)
	}
	return results, nil
}

func enclosingClass(stack []scope) string {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].kind == "class" {
			return stack[i].name
		}
	}
	return ""
}

func indentOf(line string) int {
	n := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			n++
		case '\t':
			n += 4 // treat tab as 4 spaces
		default:
			return n
		}
	}
	return n
}
