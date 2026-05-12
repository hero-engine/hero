package codescan

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// GoParser uses Go's native AST parser for accurate extraction.
type GoParser struct{}

// NewGoParser returns a parser for Go source files.
func NewGoParser() *GoParser { return &GoParser{} }

func (p *GoParser) Languages() []string { return []string{"go"} }

func (p *GoParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		// Try without comments on parse error
		f, err = parser.ParseFile(fset, path, content, 0)
		if err != nil {
			return nil, err
		}
	}

	fi := &FileInfo{
		Path:      path,
		Language:  "go",
		Package:   f.Name.Name,
		LineCount: strings.Count(string(content), "\n") + 1,
	}

	// Extract imports
	for _, imp := range f.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		fi.Imports = append(fi.Imports, impPath)
	}

	// Extract symbols
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := Symbol{
				Name:     d.Name.Name,
				Kind:     SymFunc,
				Exported: d.Name.IsExported(),
				Line:     fset.Position(d.Pos()).Line,
			}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				sym.Kind = SymMethod
				sym.Receiver = exprName(d.Recv.List[0].Type)
			}
			sym.Signature = funcSignature(d)
			if d.Doc != nil {
				sym.Doc = firstLine(d.Doc.Text())
			}
			fi.Symbols = append(fi.Symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sym := Symbol{
						Name:     s.Name.Name,
						Exported: s.Name.IsExported(),
						Line:     fset.Position(s.Pos()).Line,
					}
					switch s.Type.(type) {
					case *ast.InterfaceType:
						sym.Kind = SymInterface
					case *ast.StructType:
						sym.Kind = SymStruct
					default:
						sym.Kind = SymType
					}
					// Doc from GenDecl if single spec, else from TypeSpec
					if d.Doc != nil && len(d.Specs) == 1 {
						sym.Doc = firstLine(d.Doc.Text())
					} else if s.Comment != nil {
						sym.Doc = firstLine(s.Comment.Text())
					}
					fi.Symbols = append(fi.Symbols, sym)

				case *ast.ValueSpec:
					kind := SymVar
					if d.Tok == token.CONST {
						kind = SymConst
					}
					for _, name := range s.Names {
						sym := Symbol{
							Name:     name.Name,
							Kind:     kind,
							Exported: name.IsExported(),
							Line:     fset.Position(name.Pos()).Line,
						}
						if d.Doc != nil && len(d.Specs) == 1 {
							sym.Doc = firstLine(d.Doc.Text())
						}
						fi.Symbols = append(fi.Symbols, sym)
					}
				}
			}
		}
	}

	// Skip test files from exported symbols
	if strings.HasSuffix(filepath.Base(path), "_test.go") {
		for i := range fi.Symbols {
			fi.Symbols[i].Exported = false
		}
	}

	return fi, nil
}

func exprName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprName(t.X)
	case *ast.IndexExpr:
		return exprName(t.X)
	case *ast.IndexListExpr:
		return exprName(t.X)
	default:
		return ""
	}
}

func funcSignature(d *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func")
	if d.Recv != nil && len(d.Recv.List) > 0 {
		b.WriteString(" (" + exprName(d.Recv.List[0].Type) + ")")
	}
	b.WriteString(" " + d.Name.Name)
	b.WriteString("(")
	if d.Type.Params != nil {
		var params []string
		for _, p := range d.Type.Params.List {
			typeName := typeString(p.Type)
			if len(p.Names) == 0 {
				params = append(params, typeName)
			} else {
				for _, n := range p.Names {
					params = append(params, n.Name+" "+typeName)
				}
			}
		}
		b.WriteString(strings.Join(params, ", "))
	}
	b.WriteString(")")
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		var results []string
		for _, r := range d.Type.Results.List {
			results = append(results, typeString(r.Type))
		}
		if len(results) == 1 {
			b.WriteString(" " + results[0])
		} else {
			b.WriteString(" (" + strings.Join(results, ", ") + ")")
		}
	}
	return b.String()
}

func typeString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	case *ast.ChanType:
		return "chan " + typeString(t.Value)
	default:
		return "any"
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
