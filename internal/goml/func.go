package goml

import (
	"fmt"
	"strings"
)

// varSet accumulates free lowercase variables of a signature, in first-
// occurrence order, classified as type variables or nat index variables.
type varSet struct {
	order []string
	kind  map[string]string // "type" | "nat"
}

func newVarSet() *varSet { return &varSet{kind: map[string]string{}} }

func (v *varSet) add(name, kind string) {
	if k, ok := v.kind[name]; ok {
		if k == "type" && kind == "nat" {
			v.kind[name] = "nat" // index evidence upgrades
		}
		return
	}
	v.order = append(v.order, name)
	v.kind[name] = kind
}

// collectVars walks a type, classifying free variables. bound names are
// skipped (they are value parameters or already-declared binders).
func (c *converter) collectVars(t Type, bound map[string]bool, vs *varSet) {
	switch t := t.(type) {
	case *TypeName:
		if t.Pkg == "" && !isUpperName(t.Name) && !goPredeclared[t.Name] && !bound[t.Name] {
			if _, isBuiltin := gomlBuiltins[t.Name]; !isBuiltin {
				vs.add(t.Name, "type")
			}
		}
	case *TypeNat:
	case *TypeIndexOp:
		for _, v := range c.freeIndexVars(t, nil) {
			if !bound[v] {
				vs.add(v, "nat")
			}
		}
	case *TypeEq:
		for _, side := range []Type{t.L, t.R} {
			for _, v := range c.freeIndexVars(side, nil) {
				if !bound[v] {
					vs.add(v, "nat")
				}
			}
		}
	case *TypeArrow:
		c.collectVars(t.From, bound, vs)
		c.collectVars(t.To, bound, vs)
	case *TypeApp:
		head := t.Head
		sorts := []string(nil)
		if head.Pkg == "" {
			if info, ok := c.enums[head.Name]; ok {
				sorts = info.sorts
			}
			if head.Name == "Array" && len(t.Args) == 2 {
				sorts = []string{"nat", "any"}
			}
		}
		for i, a := range t.Args {
			if sorts != nil && i < len(sorts) && sorts[i] != "any" {
				for _, v := range c.freeIndexVars(a, nil) {
					if !bound[v] {
						vs.add(v, "nat")
					}
				}
				continue
			}
			c.collectVars(a, bound, vs)
		}
	}
}

// typeStringEq extends typeString with propositional equality.
func (c *converter) fullTypeString(t Type) string {
	if eq, ok := t.(*TypeEq); ok {
		return "Eq[" + c.indexString(eq.L) + ", " + c.indexString(eq.R) + "]"
	}
	return c.typeString(t)
}

// ------------------------------------------------------------------ lets

type fnSig struct {
	receiver   string   // "(s Stack[a])" or ""
	tparams    []string // "a any", "t Monoid"
	params     []string // "0 n nat", "v Vec[a, n+1]", "a, b int"
	result     string
	paramNames []string // clausal form: value-parameter names by column
	matchCol   int      // clausal form: column matched on (-1: none)
}

func (c *converter) printLet(d *LetDecl, ns *NamespaceDecl) {
	c.printDoc(d.Doc, "")
	tailRec := hasAttr(d.Attrs, "tail")
	for _, a := range d.Attrs {
		if a.Name == "tail" {
			continue
		}
		if line := c.directiveFor(a); line != "" {
			c.b.WriteString(line + "\n")
		}
	}
	if tailRec && !d.Rec {
		c.failf(d.Pos, "@[tail] requires `let rec`")
	}

	sig := c.buildSig(d, ns)
	kw := "func"
	if d.Total {
		kw = "total func"
	} else if tailRec {
		kw = "tail func"
	}
	var head strings.Builder
	head.WriteString(kw + " ")
	if sig.receiver != "" {
		head.WriteString(sig.receiver + " ")
	}
	head.WriteString(d.Name)
	if len(sig.tparams) > 0 {
		head.WriteString("[" + strings.Join(sig.tparams, ", ") + "]")
	}
	head.WriteString("(" + strings.Join(sig.params, ", ") + ")")
	if sig.result != "" {
		head.WriteString(" " + sig.result)
	}
	c.b.WriteString(head.String() + " {\n")

	body := d.Body
	if body == nil {
		switch {
		case sig.matchCol == -1:
			body = d.Clauses[0].Body
		default:
			clauses := make([]*Clause, len(d.Clauses))
			for i, cl := range d.Clauses {
				if cl.Row != nil {
					clauses[i] = &Clause{Alts: []Pattern{cl.Row[sig.matchCol]}, Body: cl.Body, Pos: cl.Pos}
				} else {
					clauses[i] = cl
				}
			}
			body = &Match{Subject: &Ident{Name: sig.paramNames[sig.matchCol], Pos: d.Pos}, Clauses: clauses, Pos: d.Pos}
		}
	}
	fw := &fnWriter{c: c, fnName: d.Name, tailRec: tailRec, returns: sig.result != ""}
	c.fw = fw
	fw.writeTail(body, "\t")
	c.fw = nil
	c.b.WriteString("}\n")
}

func (c *converter) buildSig(d *LetDecl, ns *NamespaceDecl) fnSig {
	var sig fnSig

	// Bound names: every value binder name.
	bound := map[string]bool{}
	for _, b := range d.Binders {
		if b.Instance {
			continue
		}
		for _, n := range b.Names {
			bound[n] = true
		}
	}

	// Declared binders claim their variables before inference runs.
	declared := map[string]bool{}
	multVars := map[string]bool{}
	type tparam struct{ name, constraint string }
	var tparams []tparam
	for _, b := range d.Binders {
		if b.Instance {
			app, ok := b.Type.(*TypeApp)
			if !ok || len(app.Args) != 1 {
				c.failf(b.Pos, "instance binders take one class applied to one variable, like [Monoid t]")
			}
			v, ok := app.Args[0].(*TypeName)
			if !ok || v.Pkg != "" || isUpperName(v.Name) {
				c.failf(b.Pos, "instance binders constrain a type variable")
			}
			tparams = append(tparams, tparam{v.Name, c.typeString(app.Head)})
			declared[v.Name] = true
			continue
		}
		if b.Implicit {
			switch c.binderSort(b) {
			case "any":
				for _, n := range b.Names {
					tparams = append(tparams, tparam{n, "any"})
					declared[n] = true
				}
			case "nat":
				for _, n := range b.Names {
					declared[n] = true
				}
			case "mult":
				for _, n := range b.Names {
					tparams = append(tparams, tparam{n, "mult"})
					declared[n] = true
					multVars[n] = true
				}
			default:
				c.failf(b.Pos, "implicit binders are Type-, Nat-, or Mult-sorted")
			}
		}
	}

	// Receiver (namespace methods).
	recvVars := map[string]bool{}
	binders := d.Binders
	if ns != nil {
		if len(binders) == 0 || binders[0].Implicit || binders[0].Instance || len(binders[0].Names) != 1 {
			c.failf(d.Pos, "a namespace let starts with one explicit receiver binder")
		}
		recv := binders[0]
		headName := ""
		switch t := recv.Type.(type) {
		case *TypeName:
			headName = t.Name
		case *TypeApp:
			headName = t.Head.Name
		}
		if headName != ns.Name {
			c.failf(recv.Pos, "receiver type %s does not match namespace %s", headName, ns.Name)
		}
		vs := newVarSet()
		c.collectVars(recv.Type, map[string]bool{}, vs)
		for _, v := range vs.order {
			recvVars[v] = true
			declared[v] = true
		}
		sig.receiver = "(" + recv.Names[0] + " " + c.typeString(recv.Type) + ")"
		binders = binders[1:]
	}

	// Inference: free signature variables not declared anywhere.
	vs := newVarSet()
	for _, b := range binders {
		if !b.Instance && !b.Implicit {
			c.collectVars(b.Type, bound, vs)
		}
	}
	if d.Result != nil {
		c.collectVars(d.Result, bound, vs)
	}
	if d.Sig != nil {
		c.collectVars(d.Sig, bound, vs)
	}
	var inferredNats []string
	for _, v := range vs.order {
		if declared[v] {
			continue
		}
		switch vs.kind[v] {
		case "type":
			tparams = append(tparams, tparam{v, "any"})
		case "nat":
			inferredNats = append(inferredNats, v)
		}
		declared[v] = true
	}

	for _, t := range tparams {
		sig.tparams = append(sig.tparams, t.name+" "+t.constraint)
	}
	for _, v := range inferredNats {
		sig.params = append(sig.params, "0 "+v+" nat")
	}
	for _, b := range binders {
		if b.Instance {
			continue // lowers to the implicit witness parameter
		}
		if b.Implicit {
			if c.binderSort(b) == "nat" {
				q := b.Quantity
				if q == "" {
					q = "0"
				}
				for _, n := range b.Names {
					sig.params = append(sig.params, q+" "+n+" nat")
				}
			}
			continue
		}
		names := b.Names
		quantity := b.Quantity
		// A multiplicity-variable quantity: (m x : a) where m is a
		// declared Mult implicit.
		if quantity == "" && len(names) >= 2 && multVars[names[0]] {
			quantity = names[0]
			names = names[1:]
		}
		typeStr := c.fullTypeString(b.Type)
		// After a multiplicity variable, composite types parenthesize
		// (the .gp grammar's instantiation ambiguity).
		if quantity != "" && quantity != "0" && quantity != "1" && !startsIdentLike(typeStr) {
			typeStr = "(" + typeStr + ")"
		}
		group := strings.Join(names, ", ") + " " + typeStr
		if quantity != "" {
			group = quantity + " " + group
		}
		sig.params = append(sig.params, group)
	}

	switch {
	case d.Sig != nil:
		c.buildClausalSig(d, &sig)
	default:
		sig.result = c.resultString(d.Result)
	}
	return sig
}

// startsIdentLike reports whether a rendered type may follow a
// multiplicity variable without parentheses in .gp.
func startsIdentLike(t string) bool {
	if t == "" {
		return false
	}
	switch t[0] {
	case '*', '(':
		return true
	}
	r := rune(t[0])
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		strings.HasPrefix(t, "func") || strings.HasPrefix(t, "chan") ||
		strings.HasPrefix(t, "map") || strings.HasPrefix(t, "struct") ||
		strings.HasPrefix(t, "interface")
}

// buildClausalSig derives value parameters from a clausal signature:
// the arrow chain names the columns, clause patterns pick the (single)
// column that is matched on.
func (c *converter) buildClausalSig(d *LetDecl, sig *fnSig) {
	var froms []Type
	t := d.Sig
	for {
		arrow, ok := t.(*TypeArrow)
		if !ok {
			break
		}
		froms = append(froms, arrow.From)
		t = arrow.To
	}
	if len(froms) == 0 {
		c.failf(d.Pos, "a clausal definition needs an arrow signature")
	}
	for _, cl := range d.Clauses {
		got := 1
		if cl.Row != nil {
			got = len(cl.Row)
		}
		if got != len(froms) {
			c.failf(cl.Pos, "clause has %d pattern(s); the signature takes %d argument(s)", got, len(froms))
		}
	}
	sig.matchCol = -1
	names := make([]string, len(froms))
	for col := range froms {
		bindName := ""
		hasCtor := false
		for _, cl := range d.Clauses {
			pats := cl.Alts
			if cl.Row != nil {
				pats = []Pattern{cl.Row[col]}
			}
			for _, p := range pats {
				switch p := p.(type) {
				case *PBind:
					if bindName == "" {
						bindName = p.Name
					} else if bindName != p.Name {
						c.failf(p.Pos, "column %d binds %s in one clause and %s in another; use one name", col+1, bindName, p.Name)
					}
				case *PCtor:
					hasCtor = true
				}
			}
		}
		if hasCtor {
			if sig.matchCol >= 0 {
				c.failf(d.Pos, "clausal definitions match on one column; destructure further with an inner match")
			}
			if bindName != "" {
				c.failf(d.Pos, "column %d mixes constructor patterns and binders across clauses", col+1)
			}
			sig.matchCol = col
		}
		if bindName == "" {
			if col == 0 {
				bindName = "v"
			} else {
				bindName = fmt.Sprintf("v%d", col)
			}
		}
		names[col] = bindName
		sig.params = append(sig.params, bindName+" "+c.fullTypeString(froms[col]))
	}
	if sig.matchCol == -1 && len(d.Clauses) > 1 {
		c.failf(d.Pos, "multiple clauses need a constructor pattern in some column")
	}
	sig.paramNames = names
	sig.result = c.resultString(t)
}

// --------------------------------------------------------- body lowering

type fnWriter struct {
	c           *converter
	fnName      string
	tailRec     bool
	returns     bool
	temps       int
	curIndent   string
	hoist       []string // pre-rendered statements to emit before the current one
	inMatchExpr bool
}

// prep sets the hoist indent before rendering a statement's expressions.
func (w *fnWriter) prep(ind string) { w.curIndent = ind }

// flush writes any statements hoisted while rendering expressions.
func (w *fnWriter) flush() {
	for _, h := range w.hoist {
		w.c.b.WriteString(h)
	}
	w.hoist = nil
}

func (w *fnWriter) writeTail(e Expr, ind string) {
	c := w.c
	c.mark(e.exprPos())
	switch e := e.(type) {
	case *Match:
		w.prep(ind)
		subj := c.exprString(e.Subject, 0)
		w.flush()
		if strings.HasPrefix(subj, "(") || strings.HasPrefix(subj, "[") || strings.HasPrefix(subj, "{") {
			w.temps++
			tmp := fmt.Sprintf("subject%d", w.temps)
			fmt.Fprintf(c.b, "%s%s := %s\n", ind, tmp, subj)
			subj = tmp
		}
		fmt.Fprintf(c.b, "%smatch %s {\n", ind, subj)
		for _, cl := range e.Clauses {
			w.writeArms(cl, ind, func(bodyInd string) { w.writeTail(cl.Body, bodyInd) })
		}
		fmt.Fprintf(c.b, "%s}\n", ind)
	case *If:
		w.prep(ind)
		cond := c.exprString(e.Cond, 0)
		w.flush()
		fmt.Fprintf(c.b, "%sif %s {\n", ind, cond)
		w.writeTail(e.Then, ind+"\t")
		fmt.Fprintf(c.b, "%s}\n", ind)
		w.writeTail(e.Else, ind)
	case *LetIn:
		lhs := ""
		assign := ":="
		switch p := e.Pat.(type) {
		case *PBind:
			lhs = p.Name
		case *PWild:
			lhs, assign = "_", "="
		default:
			c.failf(e.Pos, "let bindings destructure with match in goml v0")
		}
		switch val := e.Val.(type) {
		case *Match:
			w.writeMatchExprAssign(lhs, assign, val, ind)
		default:
			w.prep(ind)
			v := c.exprString(e.Val, 0)
			w.flush()
			fmt.Fprintf(c.b, "%s%s %s %s\n", ind, lhs, assign, v)
		}
		w.writeTail(e.Body, ind)
	case *LetStar:
		lhs, assign := "_", "="
		if pb, ok := e.Pat.(*PBind); ok {
			lhs, assign = pb.Name, ":="
		}
		w.prep(ind)
		val := c.exprString(e.Val, atomPrec)
		w.flush()
		if !strings.HasSuffix(val, "?") {
			val += "?"
		}
		fmt.Fprintf(c.b, "%s%s %s %s\n", ind, lhs, assign, val)
		w.writeTail(e.Body, ind)
	case *DoBlock:
		w.writeDoBlockTail(e, ind)
	case *SelectExpr:
		w.writeSelect(e, ind)
	case *Unit:
		fmt.Fprintf(c.b, "%sreturn\n", ind)
	default:
		if w.tailRec {
			if app, ok := e.(*App); ok {
				if id, ok := app.Fn.(*Ident); ok && id.Name == w.fnName {
					w.prep(ind)
					args := c.argList(app.Args)
					w.flush()
					fmt.Fprintf(c.b, "%srecur(%s)\n", ind, args)
					return
				}
			}
		}
		w.prep(ind)
		s := c.exprString(e, 0)
		w.flush()
		if w.returns {
			fmt.Fprintf(c.b, "%sreturn %s\n", ind, s)
			return
		}
		fmt.Fprintf(c.b, "%s%s\n", ind, s)
	}
}

// writeMatchExprAssign renders `name := match subj { case P: expr ... }`.
func (w *fnWriter) writeMatchExprAssign(lhs, assign string, m *Match, ind string) {
	c := w.c
	w.prep(ind)
	subj := c.exprString(m.Subject, 0)
	w.flush()
	fmt.Fprintf(c.b, "%s%s %s match %s {\n", ind, lhs, assign, subj)
	saved := w.inMatchExpr
	w.inMatchExpr = true
	for _, cl := range m.Clauses {
		body := cl.Body
		w.writeArms(cl, ind, func(bodyInd string) {
			fmt.Fprintf(c.b, "%s%s\n", bodyInd, c.exprString(body, 0))
		})
	}
	w.inMatchExpr = saved
	fmt.Fprintf(c.b, "%s}\n", ind)
}

// writeArms renders one clause's case line(s); or-alternatives that bind
// nothing share a multi-pattern arm, binding alternatives duplicate it.
func (w *fnWriter) writeArms(cl *Clause, ind string, body func(bodyInd string)) {
	c := w.c
	used := map[string]bool{}
	usedNames(cl.Body, used)
	if len(cl.Alts) > 1 && allNonBinding(cl.Alts) {
		pats := make([]string, len(cl.Alts))
		for i, p := range cl.Alts {
			pats[i] = c.patString(p, used)
		}
		fmt.Fprintf(c.b, "%scase %s:\n", ind, strings.Join(pats, ", "))
		body(ind + "\t")
		return
	}
	for _, p := range cl.Alts {
		fmt.Fprintf(c.b, "%scase %s:\n", ind, c.patString(p, used))
		body(ind + "\t")
	}
}

func allNonBinding(pats []Pattern) bool {
	for _, p := range pats {
		ctor, ok := p.(*PCtor)
		if !ok {
			return false
		}
		if ctor.As != "" {
			return false
		}
		for _, a := range ctor.Args {
			if _, wild := a.(*PWild); !wild {
				return false
			}
		}
	}
	return true
}

// patString renders a pattern; binders unused by the arm body print as _.
func (c *converter) patString(p Pattern, used map[string]bool) string {
	switch p := p.(type) {
	case *PWild:
		return "_"
	case *PBind:
		if used == nil || used[p.Name] {
			return p.Name
		}
		return "_"
	case *PCtor:
		name := p.Name
		if p.Pkg != "" {
			name = p.Pkg + "." + name
		}
		if p.As != "" {
			if len(p.Args) > 0 {
				c.failf(p.Pos, "`as` patterns take no field arguments in goml v0")
			}
			return p.As + " := " + name
		}
		if len(p.Args) == 0 {
			if p.Pkg == "" {
				if info := c.ctorOf(p.Name); info != nil && info.parens {
					return name + "()"
				}
			}
			return name
		}
		parts := make([]string, len(p.Args))
		for i, a := range p.Args {
			parts[i] = c.patString(a, used)
		}
		return name + "(" + strings.Join(parts, ", ") + ")"
	}
	return "_"
}

func (c *converter) ctorOf(name string) *ctorInfo {
	for _, e := range c.enums {
		if info, ok := e.ctors[name]; ok {
			return info
		}
	}
	return nil
}

// usedNames collects identifier uses in an expression.
func usedNames(e Expr, out map[string]bool) {
	switch e := e.(type) {
	case *Ident:
		out[e.Name] = true
	case *Selector:
		usedNames(e.X, out)
	case *App:
		usedNames(e.Fn, out)
		for _, a := range e.Args {
			usedNames(a, out)
		}
	case *Binop:
		usedNames(e.L, out)
		usedNames(e.R, out)
	case *Unary:
		usedNames(e.X, out)
	case *Try:
		usedNames(e.X, out)
	case *If:
		usedNames(e.Cond, out)
		usedNames(e.Then, out)
		usedNames(e.Else, out)
	case *Match:
		usedNames(e.Subject, out)
		for _, cl := range e.Clauses {
			usedNames(cl.Body, out)
		}
	case *LetIn:
		usedNames(e.Val, out)
		usedNames(e.Body, out)
	case *Fun:
		usedNames(e.Body, out)
	case *Witness:
		out[e.Name] = true
	}
}

// ----------------------------------------------------------- expressions

const atomPrec = 7

func (c *converter) exprString(e Expr, prec int) string {
	switch e := e.(type) {
	case *Ident:
		if info := c.ctorOf(e.Name); info != nil && info.parens && info.arity == 0 {
			return e.Name + "()"
		}
		if c.nullaryOps[e.Name] {
			return e.Name + "()"
		}
		return e.Name
	case *Lit:
		return e.Text
	case *Witness:
		return e.Name
	case *Selector:
		return c.exprString(e.X, atomPrec) + "." + e.Name
	case *DotSegment:
		return "." + e.Name
	case *App:
		return c.appString(e)
	case *Try:
		return c.exprString(e.X, atomPrec) + "?"
	case *Unary:
		return "-" + c.exprString(e.X, atomPrec)
	case *Binop:
		p := precOf(e.Op)
		s := c.exprString(e.L, p) + " " + e.Op + " " + c.exprString(e.R, p+1)
		if p < prec {
			return "(" + s + ")"
		}
		return s
	case *If:
		return "if " + c.exprString(e.Cond, 0) + " { " + c.exprString(e.Then, 0) + " } else { " + c.exprString(e.Else, 0) + " }"
	case *Fun:
		return c.funString(e)
	case *Match:
		// Hoist to a temporary assigned before the enclosing statement.
		if c.fw == nil || c.fw.curIndent == "" {
			c.failf(e.Pos, "match expressions need a statement to hoist before")
		}
		if c.fw.inMatchExpr {
			c.failf(e.Pos, "nested match expressions: bind the inner match with let first")
		}
		c.fw.temps++
		tmp := fmt.Sprintf("m%d", c.fw.temps)
		ind := c.fw.curIndent
		block := c.captured(func() { c.fw.writeMatchExprAssign(tmp, ":=", e, ind) })
		c.fw.hoist = append(c.fw.hoist, block)
		return tmp
	case *Unit:
		c.failf(e.Pos, "() is a call argument or function result, not a value, in goml v0")
	case *LetIn:
		c.failf(e.Pos, "let-in appears in statement position in goml v0")
	case *LetStar:
		c.failf(e.Pos, "let* appears in statement position; parenthesized let* is not supported")
	case *DoBlock:
		c.failf(e.Pos, "do blocks are statements or function bodies, not values")
	case *SelectExpr:
		c.failf(e.Pos, "select appears in statement position")
	}
	c.failf(e.exprPos(), "unsupported expression")
	return ""
}

var gpPrec = map[string]int{
	"|>": 1, ">>>": 1, ">=>": 1,
	"||": 2, "&&": 3,
	"==": 4, "!=": 4, "<": 4, "<=": 4, ">": 4, ">=": 4,
	"+": 5, "-": 5,
	"*": 6, "/": 6, "%": 6,
}

func precOf(op string) int { return gpPrec[op] }

func (c *converter) appString(e *App) string {
	head := c.exprString(e.Fn, atomPrec)
	if id, ok := e.Fn.(*Ident); ok {
		// A saturated constructor call prints as-is; the head lookup in
		// exprString would have added () to a pinned nullary ctor.
		head = id.Name
	}
	if len(e.Args) == 1 {
		if _, unit := e.Args[0].(*Unit); unit {
			return head + "()"
		}
	}
	parts := make([]string, len(e.Args))
	for i, a := range e.Args {
		if _, unit := a.(*Unit); unit {
			c.failf(a.exprPos(), "() mixes with other arguments")
		}
		parts[i] = c.exprString(a, 0)
	}
	return head + "(" + strings.Join(parts, ", ") + ")"
}

func (c *converter) argList(args []Expr) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = c.exprString(a, 0)
	}
	return strings.Join(parts, ", ")
}

func isArrow(t Type) bool {
	_, ok := t.(*TypeArrow)
	return ok
}

func (c *converter) funString(e *Fun) string {
	var params []string
	for _, b := range e.Binders {
		if b.Implicit || b.Instance || b.Quantity != "" {
			c.failf(b.Pos, "lambdas take plain typed binders in goml v0")
		}
		params = append(params, strings.Join(b.Names, ", ")+" "+c.typeString(b.Type))
	}
	res := c.resultString(e.Result)
	body := c.exprString(e.Body, 0)
	if res == "" {
		return "func(" + strings.Join(params, ", ") + ") { " + body + " }"
	}
	return "func(" + strings.Join(params, ", ") + ") " + res + " { return " + body + " }"
}
