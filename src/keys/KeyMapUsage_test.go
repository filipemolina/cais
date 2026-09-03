package keys

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoComponentShipsALibraryDefaultKeyMap enforces the rule in
// docs/DESIGN.md: a component may forward keys to a child only if that
// child's keymap was explicitly constructed.
//
// It is a source-level check rather than a per-package one on purpose. The
// keymaps live on unexported fields, so a per-package test would have to be
// remembered and written by hand for every new component - which is exactly
// the kind of discipline that decays. Walking the tree catches a component
// nobody thought to guard.
//
// The rule is enforced structurally: every list.New / viewport.New must be
// assigned to a variable, and that variable must be given a KeyMap before
// the enclosing function returns. What the map contains is the other tests
// in this package.
func TestNoComponentShipsALibraryDefaultKeyMap(t *testing.T) {
	root := filepath.Join("..", "components")

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			checkFunc(t, fset, path, fn)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
}

// checkFunc reports any list/viewport constructed in fn without a KeyMap
// assigned to it somewhere in the same function.
func checkFunc(t *testing.T, fset *token.FileSet, path string, fn *ast.FuncDecl) {
	t.Helper()

	// Receivers that were handed a KeyMap: the `vp` in `vp.KeyMap = …`.
	assigned := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range as.Lhs {
			sel, ok := lhs.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "KeyMap" {
				continue
			}
			if ident, ok := sel.X.(*ast.Ident); ok {
				assigned[ident.Name] = true
			}
		}
		return true
	})

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range as.Rhs {
			pkg, isCtor := constructorPkg(rhs)
			if !isCtor {
				continue
			}
			pos := fset.Position(as.Pos())
			if i >= len(as.Lhs) {
				t.Errorf("%s:%d: %s.New() result is not assignable, so its keymap cannot be checked", path, pos.Line, pkg)
				continue
			}
			ident, ok := as.Lhs[i].(*ast.Ident)
			if !ok {
				t.Errorf("%s:%d: %s.New() must be assigned to a plain variable so its keymap can be replaced", path, pos.Line, pkg)
				continue
			}
			if !assigned[ident.Name] {
				t.Errorf("%s:%d: %s ships %s.DefaultKeyMap() - assign %s.KeyMap from the keys package (see docs/DESIGN.md)",
					path, pos.Line, ident.Name, pkg, ident.Name)
			}
		}
		return true
	})
}

// constructorPkg reports whether expr is list.New(…) or viewport.New(…),
// and which of the two it is.
func constructorPkg(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "New" {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	switch pkg.Name {
	case "list", "viewport":
		return pkg.Name, true
	}
	return "", false
}
