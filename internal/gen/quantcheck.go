package gen

import (
	"fmt"
	"go/ast"
)

// QTT usage checking, pass 1 (v0.7.0). Declared quantities constrain
// how a parameter's identifier may appear in the body:
//
//   0 — never in a runtime (value) position; types and index terms are
//       fine, they erase.
//   1 — consumed exactly once on every path: branches must agree, use
//       inside a loop is an error, capture by a closure or go/defer
//       counts as the single consumption, and storing into a composite
//       literal is an error (the cell would outlive the discipline).
//   m — (multiplicity variable) at most once per path: admissible
//       instantiations include 0, so absence must stay legal.
//
// The checker is syntactic and path-additive: sequences add, branches
// join by agreement. Shadowing redeclarations are not tracked (v1); a
// shadowed name simply counts, erring on the loud side.

type useCount int

const (
	useNone useCount = iota
	useOne
	useMany
)

func addUse(a, b useCount) useCount {
	if a == useNone {
		return b
	}
	if b == useNone {
		return a
	}
	return useMany
}

// checkQuantities validates one function's quantity discipline.
func checkQuantities(fd *ast.FuncDecl, quantities map[string]string) []error {
	if fd.Body == nil || len(quantities) == 0 {
		return nil
	}
	var errs []error
	for name, q := range quantities {
		switch q {
		case "0":
			if pos, found := runtimeUse(fd.Body, name); found {
				errs = append(errs, fmt.Errorf("parameter %s has quantity 0: it exists only at check time and cannot be used at runtime (types and index terms are fine)%s", name, pos))
			}
		case "1":
			n, err := pathUses(fd.Body.List, name, false)
			if err != nil {
				errs = append(errs, fmt.Errorf("linear parameter %s: %v", name, err))
			} else if n == useNone {
				errs = append(errs, fmt.Errorf("linear parameter %s is never consumed; a quantity-1 value must be used exactly once on every path", name))
			} else if n == useMany {
				errs = append(errs, fmt.Errorf("linear parameter %s is consumed more than once on some path; a quantity-1 value must be used exactly once", name))
			}
		default: // multiplicity variable: at most once per path
			n, err := pathUses(fd.Body.List, name, true)
			if err != nil {
				errs = append(errs, fmt.Errorf("parameter %s (multiplicity %s): %v", name, q, err))
			} else if n == useMany {
				errs = append(errs, fmt.Errorf("parameter %s has multiplicity %s, which admits 0: it may be used at most once on any path, but some path consumes it more than once", name, q))
			}
		}
	}
	return errs
}

// runtimeUse finds a value-position occurrence of name.
func runtimeUse(body *ast.BlockStmt, name string) (string, bool) {
	found := false
	var walk func(n ast.Node) bool
	walk = func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ValueSpec:
			// The declared TYPE erases; initial values are runtime.
			for _, v := range x.Values {
				ast.Inspect(v, walk)
			}
			return false
		case *ast.CompositeLit:
			for _, e := range x.Elts {
				ast.Inspect(e, walk)
			}
			return false // the Type erases
		case *ast.TypeAssertExpr:
			ast.Inspect(x.X, walk)
			return false
		case *ast.IndexExpr, *ast.IndexListExpr:
			// Could be an instantiation whose index args erase; value
			// indexing of a 0-param base would also name it first, so
			// inspect only the base.
			switch ie := n.(type) {
			case *ast.IndexExpr:
				ast.Inspect(ie.X, walk)
			case *ast.IndexListExpr:
				ast.Inspect(ie.X, walk)
			}
			return false
		case *ast.Ident:
			if x.Name == name {
				found = true
			}
		}
		return true
	}
	ast.Inspect(body, walk)
	return "", found
}

// pathUses computes how often name is consumed along any path of a
// statement list. atMost relaxes the every-path requirement (mult
// variables admit absence).
func pathUses(stmts []ast.Stmt, name string, atMost bool) (useCount, error) {
	total := useNone
	for i, s := range stmts {
		// Early-exit modeling: `if cond { …; return }` splits the paths —
		// the tail belongs to the fallthrough path only.
		if ifs, isIf := s.(*ast.IfStmt); isIf && ifs.Else == nil && endsInReturn(ifs.Body.List) {
			condN := exprUses(ifs.Cond, name)
			thenN, err := pathUses(ifs.Body.List, name, atMost)
			if err != nil {
				return useMany, err
			}
			restN, err := pathUses(stmts[i+1:], name, atMost)
			if err != nil {
				return useMany, err
			}
			if !atMost && thenN != restN {
				return useMany, fmt.Errorf("the early-return path and the fallthrough path consume it a different number of times; every path must consume a linear value exactly once")
			}
			branch := thenN
			if restN > branch {
				branch = restN
			}
			return addUse(total, addUse(condN, branch)), nil
		}
		n, err := stmtUses(s, name, atMost)
		if err != nil {
			return useMany, err
		}
		total = addUse(total, n)
	}
	return total, nil
}

// endsInReturn reports whether a statement list's last statement exits.
func endsInReturn(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	switch last := stmts[len(stmts)-1].(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.ExprStmt:
		if call, ok := last.X.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" {
				return true
			}
		}
	}
	return false
}

func stmtUses(s ast.Stmt, name string, atMost bool) (useCount, error) {
	switch x := s.(type) {
	case *ast.IfStmt:
		n := exprUses(x.Cond, name)
		thenN, err := pathUses(x.Body.List, name, atMost)
		if err != nil {
			return useMany, err
		}
		var elseN useCount
		switch e := x.Else.(type) {
		case *ast.BlockStmt:
			elseN, err = pathUses(e.List, name, atMost)
		case *ast.IfStmt:
			elseN, err = stmtUses(e, name, atMost)
		case nil:
			elseN = useNone
		}
		if err != nil {
			return useMany, err
		}
		if !atMost && thenN != elseN {
			return useMany, fmt.Errorf("the branches of an if consume it a different number of times; every path must consume a linear value exactly once")
		}
		branch := thenN
		if elseN > branch {
			branch = elseN
		}
		return addUse(n, branch), nil
	case *ast.ForStmt, *ast.RangeStmt:
		var body *ast.BlockStmt
		if f, ok := x.(*ast.ForStmt); ok {
			body = f.Body
		} else {
			body = x.(*ast.RangeStmt).Body
		}
		inner, err := pathUses(body.List, name, true)
		if err != nil {
			return useMany, err
		}
		if inner != useNone {
			return useMany, fmt.Errorf("it is consumed inside a loop, which may run any number of times")
		}
		return stmtExprUses(x, name, body), nil
	case *ast.BlockStmt:
		return pathUses(x.List, name, atMost)
	default:
		n := useNone
		ast.Inspect(s, func(m ast.Node) bool {
			switch y := m.(type) {
			case *ast.FuncLit:
				// Capture consumes once; uses inside are the closure's
				// own affair (it owns the value now).
				captured := false
				ast.Inspect(y.Body, func(k ast.Node) bool {
					if id, ok := k.(*ast.Ident); ok && id.Name == name {
						captured = true
					}
					return true
				})
				if captured {
					n = addUse(n, useOne)
				}
				return false
			case *ast.Ident:
				if y.Name == name {
					n = addUse(n, useOne)
				}
			}
			return true
		})
		return n, nil
	}
}

// exprUses counts occurrences in one expression.
func exprUses(e ast.Expr, name string) useCount {
	if e == nil {
		return useNone
	}
	n := useNone
	ast.Inspect(e, func(m ast.Node) bool {
		if id, ok := m.(*ast.Ident); ok && id.Name == name {
			n = addUse(n, useOne)
		}
		return true
	})
	return n
}

// stmtExprUses counts uses in a loop statement OUTSIDE its body (the
// init/cond/post/range expression).
func stmtExprUses(s ast.Stmt, name string, body *ast.BlockStmt) useCount {
	n := useNone
	ast.Inspect(s, func(m ast.Node) bool {
		if m == body {
			return false
		}
		if id, ok := m.(*ast.Ident); ok && id.Name == name {
			n = addUse(n, useOne)
		}
		return true
	})
	return n
}
