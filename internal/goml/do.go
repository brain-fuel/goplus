package goml

import (
	"fmt"
	"strings"
)

// Do-block lowering: a DoBlock in tail position runs its statements and
// treats a final bare expression as the function result; nested blocks
// (loop bodies, select arms) lower every statement plainly.

func (w *fnWriter) writeDoBlockTail(b *DoBlock, ind string) {
	for i, st := range b.Stmts {
		last := i == len(b.Stmts)-1
		if last {
			if es, ok := st.(*DoExprStmt); ok {
				switch es.X.(type) {
				case *Unit, *SelectExpr:
					w.writeDoStmt(st, ind)
				default:
					w.writeTail(es.X, ind)
				}
				continue
			}
		}
		w.writeDoStmt(st, ind)
	}
}

func (w *fnWriter) writeDoStmt(st DoStmt, ind string) {
	c := w.c
	c.mark(st.stmtPos())
	switch st := st.(type) {
	case *DoLet:
		lhs := strings.Join(st.Names, ", ")
		if val, ok := st.Val.(*Match); ok && len(st.Names) == 1 {
			w.writeMatchExprAssign(lhs, ":=", val, ind)
			return
		}
		w.prep(ind)
		v := c.exprWant(st.Val, 0, st.Type)
		w.flush()
		if allWild(st.Names) {
			fmt.Fprintf(c.b, "%s%s = %s\n", ind, lhs, v)
			return
		}
		fmt.Fprintf(c.b, "%s%s := %s\n", ind, lhs, v)
	case *DoAssign:
		w.prep(ind)
		target := c.exprString(st.Target, atomPrec)
		v := c.exprString(st.Val, 0)
		w.flush()
		fmt.Fprintf(c.b, "%s%s = %s\n", ind, target, v)
	case *DoWhile:
		w.prep(ind)
		cond := c.exprString(st.Cond, 0)
		if len(w.hoist) > 0 {
			c.failf(st.Pos, "a while condition cannot contain a match expression (it would evaluate once); bind it inside the loop")
		}
		if cond == "true" {
			fmt.Fprintf(c.b, "%sfor {\n", ind)
		} else {
			fmt.Fprintf(c.b, "%sfor %s {\n", ind, cond)
		}
		w.writeDoStmts(st.Body, ind+"\t")
		fmt.Fprintf(c.b, "%s}\n", ind)
	case *DoFor:
		w.prep(ind)
		seq := c.exprString(st.Seq, 0)
		w.flush()
		fmt.Fprintf(c.b, "%sfor %s := range %s {\n", ind, strings.Join(st.Names, ", "), seq)
		w.writeDoStmts(st.Body, ind+"\t")
		fmt.Fprintf(c.b, "%s}\n", ind)
	case *DoSend:
		w.prep(ind)
		ch := c.exprString(st.Chan, atomPrec)
		val := c.exprString(st.Val, 0)
		w.flush()
		fmt.Fprintf(c.b, "%s%s <- %s\n", ind, ch, val)
	case *DoDefer:
		w.prep(ind)
		call := c.exprString(st.Call, 0)
		w.flush()
		fmt.Fprintf(c.b, "%sdefer %s\n", ind, call)
	case *DoGo:
		w.prep(ind)
		call := c.exprString(st.Call, 0)
		w.flush()
		fmt.Fprintf(c.b, "%sgo %s\n", ind, call)
	case *DoReturn:
		if st.Val == nil {
			fmt.Fprintf(c.b, "%sreturn\n", ind)
			return
		}
		w.prep(ind)
		v := c.exprWant(st.Val, 0, c.retWant)
		w.flush()
		fmt.Fprintf(c.b, "%sreturn %s\n", ind, v)
	case *DoExprStmt:
		switch x := st.X.(type) {
		case *Unit:
			// explicit no-op
		case *SelectExpr:
			w.writeSelect(x, ind)
		case *Match:
			w.writeStmtMatch(x, ind)
		case *DoBlock:
			w.writeDoStmts(x, ind)
		case *If:
			w.writeStmtExpr(x, ind)
		default:
			w.prep(ind)
			s := c.exprString(st.X, 0)
			w.flush()
			fmt.Fprintf(c.b, "%s%s\n", ind, s)
		}
	}
}

// writeDoStmts lowers a nested block (loop body, arm body) with no
// tail-position result handling.
func (w *fnWriter) writeDoStmts(b *DoBlock, ind string) {
	for _, st := range b.Stmts {
		w.writeDoStmt(st, ind)
	}
}

// writeStmtMatch lowers a match used as a statement (arms for effect).
func (w *fnWriter) writeStmtMatch(m *Match, ind string) {
	c := w.c
	w.prep(ind)
	subj := c.exprString(m.Subject, 0)
	w.flush()
	fmt.Fprintf(c.b, "%smatch %s {\n", ind, subj)
	for _, cl := range m.Clauses {
		body := cl.Body
		w.writeArms(cl, ind, func(bodyInd string) {
			w.writeStmtExpr(body, bodyInd)
		})
	}
	fmt.Fprintf(c.b, "%s}\n", ind)
}

// writeStmtExpr lowers an expression in statement position.
func (w *fnWriter) writeStmtExpr(e Expr, ind string) {
	c := w.c
	switch e := e.(type) {
	case *Unit:
	case *DoBlock:
		w.writeDoStmts(e, ind)
	case *SelectExpr:
		w.writeSelect(e, ind)
	case *Match:
		w.writeStmtMatch(e, ind)
	case *If:
		w.prep(ind)
		cond := c.exprString(e.Cond, 0)
		w.flush()
		fmt.Fprintf(c.b, "%sif %s {\n", ind, cond)
		w.writeStmtExpr(e.Then, ind+"\t")
		// An empty else is how goml spells "no else", since `if` is an
		// expression and always has both arms.
		if isEmptyBranch(e.Else) {
			fmt.Fprintf(c.b, "%s}\n", ind)
			return
		}
		fmt.Fprintf(c.b, "%s} else {\n", ind)
		w.writeStmtExpr(e.Else, ind+"\t")
		fmt.Fprintf(c.b, "%s}\n", ind)
	default:
		w.prep(ind)
		s := c.exprString(e, 0)
		w.flush()
		fmt.Fprintf(c.b, "%s%s\n", ind, s)
	}
}

// writeSelect lowers `select with` to a native Go select statement.
func (w *fnWriter) writeSelect(sel *SelectExpr, ind string) {
	c := w.c
	c.mark(sel.Pos)
	fmt.Fprintf(c.b, "%sselect {\n", ind)
	for _, arm := range sel.Arms {
		switch arm.Kind {
		case "recv":
			ch := c.exprString(arm.Chan, atomPrec)
			w.noHoist(arm.Pos)
			if pb, ok := arm.Pat.(*PBind); ok {
				fmt.Fprintf(c.b, "%scase %s := <-%s:\n", ind, pb.Name, ch)
			} else {
				fmt.Fprintf(c.b, "%scase <-%s:\n", ind, ch)
			}
		case "send":
			ch := c.exprString(arm.Chan, atomPrec)
			val := c.exprString(arm.Val, 0)
			w.noHoist(arm.Pos)
			fmt.Fprintf(c.b, "%scase %s <- %s:\n", ind, ch, val)
		case "default":
			fmt.Fprintf(c.b, "%sdefault:\n", ind)
		}
		w.writeDoStmt(arm.Body, ind+"\t")
	}
	fmt.Fprintf(c.b, "%s}\n", ind)
}

// noHoist rejects match expressions in positions Go's grammar cannot
// host a preceding statement (select communication clauses).
func (w *fnWriter) noHoist(pos Pos) {
	if len(w.hoist) > 0 {
		w.c.failf(pos, "select arms cannot contain match expressions; bind before the select")
	}
}

// isEmptyBranch reports whether a branch does nothing: `do { }` or `()`.
func isEmptyBranch(e Expr) bool {
	switch e := e.(type) {
	case *Unit:
		return true
	case *DoBlock:
		return len(e.Stmts) == 0
	}
	return false
}

func allWild(names []string) bool {
	for _, n := range names {
		if n != "_" {
			return false
		}
	}
	return true
}
