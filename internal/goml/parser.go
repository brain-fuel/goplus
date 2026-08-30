package goml

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Parse parses one .goml source file.
func Parse(path string, src []byte) (f *File, err error) {
	toks, lerr := newLexer(path, src).tokens()
	if lerr != nil {
		return nil, lerr
	}
	p := &parser{path: path, toks: toks}
	defer func() {
		if r := recover(); r != nil {
			b, ok := r.(bailout)
			if !ok {
				panic(r)
			}
			f, err = nil, b.err
		}
	}()
	return p.parseFile(), nil
}

type bailout struct{ err *Error }

type parser struct {
	path string
	toks []Token
	i    int

	// Trailing "--" comment block, for doc attachment.
	docBlock []string
	docEnd   int // line the block ends on
}

func (p *parser) fail(pos Pos, format string, args ...any) {
	panic(bailout{&Error{Path: p.path, Pos: pos, Msg: fmt.Sprintf(format, args...)}})
}

// tok returns the current significant token, folding comment runs into
// the pending doc block.
func (p *parser) tok() Token {
	for p.toks[p.i].Kind == COMMENT {
		c := p.toks[p.i]
		if c.Pos.Line == p.docEnd+1 {
			p.docBlock = append(p.docBlock, c.Text)
		} else {
			p.docBlock = []string{c.Text}
		}
		p.docEnd = c.Pos.Line
		p.i++
	}
	return p.toks[p.i]
}

func (p *parser) at(k Kind) bool { return p.tok().Kind == k }

func (p *parser) accept(k Kind) (Token, bool) {
	if p.at(k) {
		t := p.tok()
		p.i++
		return t, true
	}
	return Token{}, false
}

func (p *parser) expect(k Kind, what string) Token {
	t, ok := p.accept(k)
	if !ok {
		got := p.tok()
		text := got.Text
		if got.Kind == EOF {
			text = "end of file"
		}
		p.fail(got.Pos, "expected %s, found %q", what, text)
	}
	return t
}

// docFor claims the pending doc block when it ends directly above line.
func (p *parser) docFor(line int) []string {
	if p.docEnd == line-1 && len(p.docBlock) > 0 {
		d := p.docBlock
		p.docBlock, p.docEnd = nil, 0
		return d
	}
	return nil
}

func isUpperName(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// ------------------------------------------------------------------ file

func (p *parser) parseFile() *File {
	f := &File{}
	for p.at(At) && p.toks[p.i+1].Kind == LBrack {
		f.Attrs = append(f.Attrs, p.parseAttrs()...)
	}
	p.expect(KwModule, "`module`")
	f.Module = p.expect(IDENT, "module name").Text
	for {
		if _, ok := p.accept(KwImport); ok {
			imp := &Import{Path: p.expect(STRING, "import path").Text}
			if _, ok := p.accept(KwAs); ok {
				imp.Alias = p.expect(IDENT, "import alias").Text
			}
			f.Imports = append(f.Imports, imp)
			continue
		}
		if t, ok := p.accept(KwOpen); ok {
			p.fail(t.Pos, "`open` is not supported in goml v0; use qualified names")
		}
		break
	}
	for !p.at(EOF) {
		f.Decls = append(f.Decls, p.parseDecl())
	}
	return f
}

func (p *parser) parseDecl() Decl {
	var attrs []Attr
	for p.at(At) && p.toks[p.i+1].Kind == LBrack {
		attrs = append(attrs, p.parseAttrs()...)
	}
	t := p.tok()
	doc := p.docFor(t.Pos.Line)
	switch t.Kind {
	case KwTotal:
		p.i++
		p.expect(KwLet, "`let` after `total`")
		return p.parseLet(doc, attrs, true)
	case KwLet:
		p.i++
		return p.parseLet(doc, attrs, false)
	case KwType:
		p.i++
		return p.parseTypeDecl(doc, attrs)
	case KwClass:
		p.i++
		return p.parseClass(doc, attrs)
	case KwInstance:
		p.i++
		return p.parseInstance(doc, attrs)
	case KwNamespace:
		p.i++
		return p.parseNamespace()
	case KwDo, KwSelect, KwWhile, KwFor, KwLetStar:
		p.fail(t.Pos, "%q is reserved but not supported in goml v0", t.Text)
	}
	p.fail(t.Pos, "expected a declaration, found %q", t.Text)
	return nil
}

func (p *parser) parseAttrs() []Attr {
	p.expect(At, "`@`")
	p.expect(LBrack, "`[` after `@`")
	var out []Attr
	for {
		name := p.expect(IDENT, "attribute name")
		attr := Attr{Name: name.Text, Pos: name.Pos}
		for p.at(IDENT) || p.at(STRING) || p.at(INT) || p.at(FLOAT) {
			t := p.tok()
			p.i++
			attr.Args = append(attr.Args, t.Text)
		}
		out = append(out, attr)
		if _, ok := p.accept(Comma); !ok {
			break
		}
	}
	p.expect(RBrack, "`]` closing attribute")
	return out
}

// ------------------------------------------------------------------- let

func (p *parser) parseLet(doc []string, attrs []Attr, total bool) *LetDecl {
	d := &LetDecl{Doc: doc, Attrs: attrs, Total: total}
	if _, ok := p.accept(KwRec); ok {
		d.Rec = true
	}
	name := p.expect(IDENT, "binding name")
	d.Name, d.Pos = name.Text, name.Pos
	for p.at(LParen) || p.at(LBrace) || p.at(LBrack) {
		d.Binders = append(d.Binders, p.parseBinder())
	}
	if _, ok := p.accept(Colon); ok {
		t := p.parseType()
		if p.at(Bar) {
			if len(d.Binders) > 0 {
				p.fail(name.Pos, "clausal definitions take no binders in goml v0; move them into the signature type")
			}
			d.Sig = t
			d.Clauses = p.parseClauses()
			d.Where = p.parseWhereHelpers()
			return d
		}
		d.Result = t
	}
	p.expect(Assign, "`:=`")
	d.Body = p.parseExpr()
	d.Where = p.parseWhereHelpers()
	return d
}

// parseWhereHelpers parses an optional trailing helper block:
// `where h (x : T) : R := e; g (y : T) : R := e`. Helpers are
// package-private (lowercase) and closed over nothing.
func (p *parser) parseWhereHelpers() []*LetDecl {
	if _, ok := p.accept(KwWhere); !ok {
		return nil
	}
	var out []*LetDecl
	for {
		name := p.expect(IDENT, "helper name after `where`")
		if isUpperName(name.Text) {
			p.fail(name.Pos, "where-helpers are package-private; %q must be lowercase", name.Text)
		}
		h := &LetDecl{Name: name.Text, Pos: name.Pos, Doc: p.docFor(name.Pos.Line)}
		for p.at(LParen) || p.at(LBrace) || p.at(LBrack) {
			h.Binders = append(h.Binders, p.parseBinder())
		}
		if _, ok := p.accept(Colon); ok {
			h.Result = p.parseType()
		}
		p.expect(Assign, "`:=` in where-helper")
		h.Body = p.parseExpr()
		out = append(out, h)
		if _, ok := p.accept(Semi); !ok {
			return out
		}
	}
}

func (p *parser) parseBinder() *Binder {
	t := p.tok()
	b := &Binder{Pos: t.Pos}
	var closer Kind
	switch t.Kind {
	case LParen:
		// The unit binder: `let main () := ...` is a nullary function,
		// distinguishing it from `let X := ...`, which binds a value.
		if p.toks[p.i+1].Kind == RParen {
			p.i += 2
			b.Unit = true
			return b
		}
		closer = RParen
	case LBrace:
		closer = RBrace
		b.Implicit = true
	case LBrack:
		p.i++
		b.Instance = true
		b.Type = p.parseTypeApp()
		p.expect(RBrack, "`]` closing instance binder")
		return b
	}
	p.i++
	if q, ok := p.accept(INT); ok {
		if q.Text != "0" && q.Text != "1" {
			p.fail(q.Pos, "quantity must be 0 or 1, found %q", q.Text)
		}
		b.Quantity = q.Text
	}
	for p.at(IDENT) {
		b.Names = append(b.Names, p.tok().Text)
		p.i++
	}
	if len(b.Names) == 0 {
		p.fail(p.tok().Pos, "expected binder name")
	}
	p.expect(Colon, "`:` in binder")
	b.Type = p.parseType()
	p.expect(closer, "closing bracket of binder")
	return b
}

func (p *parser) parseClauses() []*Clause {
	var out []*Clause
	for p.at(Bar) {
		bar := p.tok()
		c := &Clause{Pos: bar.Pos}
		p.expect(Bar, "`|`")
		first := p.parsePattern()
		if p.at(Comma) {
			// Multi-column row: | 0, Cons h t => ...
			c.Row = []Pattern{first}
			for {
				if _, ok := p.accept(Comma); !ok {
					break
				}
				c.Row = append(c.Row, p.parsePattern())
			}
		} else {
			c.Alts = []Pattern{first}
			for !p.at(FatArrow) {
				if !p.at(Bar) {
					p.fail(p.tok().Pos, "expected `|` or `=>` in match clause, found %q", p.tok().Text)
				}
				p.expect(Bar, "`|`")
				c.Alts = append(c.Alts, p.parsePattern())
			}
		}
		p.expect(FatArrow, "`=>`")
		c.Body = p.parseExpr()
		out = append(out, c)
	}
	if len(out) == 0 {
		p.fail(p.tok().Pos, "expected at least one `|` clause")
	}
	return out
}

// -------------------------------------------------------------- patterns

func (p *parser) parsePattern() Pattern {
	t := p.tok()
	switch t.Kind {
	case LParen:
		p.i++
		inner := p.parsePattern()
		p.expect(RParen, "`)` closing pattern")
		return inner
	case IDENT:
		p.i++
		if t.Text == "_" {
			return &PWild{Pos: t.Pos}
		}
		if !isUpperName(t.Text) {
			// A lowercase name followed by `.Ctor` is an imported
			// constructor (result.Ok v); alone it binds.
			if !p.at(Dot) || !isUpperName(p.toks[p.i+1].Text) {
				return &PBind{Name: t.Text, Pos: t.Pos}
			}
			p.i++
			return p.parseCtorPattern(&PCtor{Pkg: t.Text, Name: p.expect(IDENT, "constructor name").Text, Pos: t.Pos})
		}
		ctor := &PCtor{Name: t.Text, Pos: t.Pos}
		if _, ok := p.accept(Dot); ok {
			ctor.Pkg = ctor.Name
			ctor.Name = p.expect(IDENT, "constructor name").Text
		}
		return p.parseCtorPattern(ctor)
	}
	p.fail(t.Pos, "expected a pattern, found %q", t.Text)
	return nil
}

// parseCtorPattern reads a constructor pattern's arguments and optional
// `as` binder, having already consumed its name.
func (p *parser) parseCtorPattern(ctor *PCtor) Pattern {
	{
		for {
			a := p.tok()
			if a.Kind == IDENT && a.Text == "_" {
				p.i++
				ctor.Args = append(ctor.Args, &PWild{Pos: a.Pos})
				continue
			}
			if a.Kind == IDENT && !isUpperName(a.Text) {
				p.i++
				ctor.Args = append(ctor.Args, &PBind{Name: a.Text, Pos: a.Pos})
				continue
			}
			if a.Kind == IDENT && isUpperName(a.Text) {
				p.i++
				ctor.Args = append(ctor.Args, &PCtor{Name: a.Text, Pos: a.Pos})
				continue
			}
			if a.Kind == LParen {
				p.i++
				ctor.Args = append(ctor.Args, p.parsePattern())
				p.expect(RParen, "`)` closing pattern")
				continue
			}
			break
		}
		if _, ok := p.accept(KwAs); ok {
			ctor.As = p.expect(IDENT, "binder after `as`").Text
		}
		return ctor
	}
}

// ----------------------------------------------------------------- types

func (p *parser) parseType() Type {
	t := p.parseTypeIndex()
	if eq, ok := p.accept(Eq); ok {
		return &TypeEq{L: t, R: p.parseTypeIndex(), Pos: eq.Pos}
	}
	if arrow, ok := p.accept(Arrow); ok {
		return &TypeArrow{From: t, To: p.parseType(), Pos: arrow.Pos}
	}
	return t
}

func (p *parser) parseTypeIndex() Type {
	t := p.parseTypeApp()
	for {
		tk := p.tok()
		var op string
		switch tk.Kind {
		case Plus:
			op = "+"
		case Minus:
			op = "-"
		case Star:
			op = "*"
		default:
			return t
		}
		p.i++
		t = &TypeIndexOp{Op: op, L: t, R: p.parseTypeApp(), Pos: tk.Pos}
	}
}

func (p *parser) parseTypeApp() Type {
	head := p.parseTypeAtom()
	name, ok := head.(*TypeName)
	if !ok {
		return head
	}
	var args []Type
	for {
		switch p.tok().Kind {
		case IDENT, INT, LParen:
			args = append(args, p.parseTypeAtom())
		default:
			if len(args) == 0 {
				return head
			}
			return &TypeApp{Head: name, Args: args, Pos: name.Pos}
		}
	}
}

func (p *parser) parseTypeAtom() Type {
	t := p.tok()
	switch t.Kind {
	case IDENT:
		p.i++
		name := &TypeName{Name: t.Text, Pos: t.Pos}
		if p.at(Dot) && p.toks[p.i+1].Kind == IDENT {
			p.i++
			name.Pkg = name.Name
			name.Name = p.expect(IDENT, "qualified type name").Text
		}
		return name
	case INT:
		p.i++
		return &TypeNat{Lit: t.Text, Pos: t.Pos}
	case LParen:
		p.i++
		inner := p.parseType()
		p.expect(RParen, "`)` closing type")
		return inner
	}
	p.fail(t.Pos, "expected a type, found %q", t.Text)
	return nil
}

// ------------------------------------------------------------ type decls

func (p *parser) parseTypeDecl(doc []string, attrs []Attr) *TypeDecl {
	name := p.expect(IDENT, "type name")
	d := &TypeDecl{Doc: doc, Attrs: attrs, Name: name.Text, Pos: name.Pos}
	for p.at(LParen) || p.at(LBrace) {
		d.Binders = append(d.Binders, p.parseBinder())
	}
	if _, ok := p.accept(Colon); ok {
		// Kind: S1 -> S2 -> ... -> Type. A `Type` followed by `->` is a
		// type-sorted parameter (`Expr : Type -> Type`); the final
		// `Type` terminates.
		for {
			s := p.parseTypeApp()
			if n, ok := s.(*TypeName); ok && n.Pkg == "" && n.Name == "Type" && !p.at(Arrow) {
				break
			}
			d.Kind = append(d.Kind, s)
			p.expect(Arrow, "`->` in kind")
		}
	}
	if _, ok := p.accept(KwWhere); ok {
		d.Where = true
		d.Sum = p.parseCtors(true)
		d.Deriving = p.parseDeriving()
		return d
	}
	p.expect(Assign, "`:=` or `where`")
	switch p.tok().Kind {
	case Bar:
		d.Sum = p.parseCtors(false)
	case LBrace:
		p.parseRecordOrRefine(d)
	case KwInterface:
		p.i++
		if len(d.Kind) > 0 {
			p.fail(d.Pos, "an interface declaration takes binders, not a kind")
		}
		d.Iface = p.parseIfaceBody()
	case KwProp:
		p.i++
		if len(d.Kind) > 0 {
			p.fail(d.Pos, "a prop declaration takes binders, not a kind")
		}
		for _, b := range d.Binders {
			if b.Implicit {
				p.fail(b.Pos, "prop parameters are explicit `(i : Nat)` binders")
			}
		}
		p.expect(LBrace, "`{` opening the proposition")
		d.Prop = p.parseType()
		p.expect(RBrace, "`}` closing the proposition")
	default:
		d.Alias = p.parseType()
	}
	d.Deriving = p.parseDeriving()
	if d.Prop != nil && len(d.Deriving) > 0 {
		p.fail(d.Pos, "a prop declaration names facts, not values; it cannot derive")
	}
	if d.Iface != nil && len(d.Deriving) > 0 {
		p.fail(d.Pos, "an interface declaration cannot derive")
	}
	return d
}

// parseIfaceBody parses `{ Name : Sig; …; Embedded }`. A member that is
// a bare (possibly qualified) type name is an embedded interface.
func (p *parser) parseIfaceBody() *IfaceBody {
	p.expect(LBrace, "`{` opening the interface")
	body := &IfaceBody{}
	for !p.at(RBrace) {
		name := p.expect(IDENT, "interface member")
		m := &IfaceMember{Pos: name.Pos, Doc: p.docFor(name.Pos.Line)}
		if _, ok := p.accept(Colon); ok {
			m.Name = name.Text
			m.Sig = p.parseType()
		} else if _, ok := p.accept(Dot); ok {
			sel := p.expect(IDENT, "embedded interface name")
			m.Sig = &TypeName{Pkg: name.Text, Name: sel.Text, Pos: name.Pos}
		} else {
			m.Sig = &TypeName{Name: name.Text, Pos: name.Pos}
		}
		body.Members = append(body.Members, m)
		if _, ok := p.accept(Semi); !ok {
			break
		}
	}
	p.expect(RBrace, "`}` closing the interface")
	return body
}

func (p *parser) parseDeriving() []string {
	if _, ok := p.accept(KwDeriving); !ok {
		return nil
	}
	var out []string
	for {
		out = append(out, p.expect(IDENT, "deriving name").Text)
		if _, ok := p.accept(Comma); !ok {
			return out
		}
	}
}

func (p *parser) parseCtors(where bool) []*Ctor {
	var out []*Ctor
	for p.at(Bar) {
		p.i++
		name := p.expect(IDENT, "constructor name")
		if !isUpperName(name.Text) {
			p.fail(name.Pos, "constructor names are capitalized; %q is not", name.Text)
		}
		c := &Ctor{Name: name.Text, Pos: name.Pos, Doc: p.docFor(name.Pos.Line)}
		for p.at(LParen) || p.at(LBrace) {
			b := p.parseBinder()
			if b.Implicit {
				c.Exist = append(c.Exist, b)
				continue
			}
			for _, n := range b.Names {
				c.Fields = append(c.Fields, &Field{Name: n, Type: b.Type, Pos: b.Pos})
			}
		}
		if _, ok := p.accept(Colon); ok {
			c.Result = p.parseType()
		} else if where {
			p.fail(name.Pos, "constructor %s needs a `: Result` signature in a where-form type", c.Name)
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		p.fail(p.tok().Pos, "expected at least one `|` constructor")
	}
	return out
}

func (p *parser) parseRecordOrRefine(d *TypeDecl) {
	p.expect(LBrace, "`{`")
	first := p.expect(IDENT, "field name")
	p.expect(Colon, "`:` in field")
	firstType := p.parseType()
	if _, ok := p.accept(Bar); ok {
		d.Refine = &Refine{Binder: first.Text, Base: firstType, Pred: p.parseExpr()}
		p.expect(RBrace, "`}` closing refinement")
		return
	}
	f := &Field{Name: first.Text, Type: firstType, Pos: first.Pos}
	f.Attrs = p.parseFieldAttrs()
	d.Record = append(d.Record, f)
	for {
		if _, ok := p.accept(Semi); !ok {
			break
		}
		if p.at(RBrace) {
			break
		}
		name := p.expect(IDENT, "field name")
		p.expect(Colon, "`:` in field")
		ft := p.parseType()
		nf := &Field{Name: name.Text, Type: ft, Pos: name.Pos}
		nf.Attrs = p.parseFieldAttrs()
		d.Record = append(d.Record, nf)
	}
	p.expect(RBrace, "`}` closing record")
}

func (p *parser) parseFieldAttrs() []Attr {
	var out []Attr
	for p.at(At) && p.toks[p.i+1].Kind == LBrack {
		out = append(out, p.parseAttrs()...)
	}
	return out
}

// --------------------------------------------------------------- classes

func (p *parser) parseClass(doc []string, attrs []Attr) *ClassDecl {
	name := p.expect(IDENT, "class name")
	d := &ClassDecl{Doc: doc, Attrs: attrs, Name: name.Text, Pos: name.Pos}
	if !p.at(LParen) {
		p.fail(p.tok().Pos, "class %s needs exactly one `(t : Type)` binder", d.Name)
	}
	d.Binder = p.parseBinder()
	if _, ok := p.accept(KwExtends); ok {
		for {
			d.Extends = append(d.Extends, p.parseTypeApp())
			if _, ok := p.accept(Comma); !ok {
				break
			}
		}
	}
	p.expect(KwWhere, "`where`")
	for {
		if _, ok := p.accept(KwLaw); ok {
			lname := p.expect(IDENT, "law name")
			law := &Law{Name: lname.Text, Pos: lname.Pos, Doc: p.docFor(lname.Pos.Line)}
			for p.at(LParen) {
				law.Binders = append(law.Binders, p.parseBinder())
			}
			p.expect(Assign, "`:=` in law")
			law.Body = p.parseExpr()
			d.Laws = append(d.Laws, law)
			continue
		}
		if !p.at(IDENT) {
			break
		}
		oname := p.expect(IDENT, "operation name")
		op := &ClassOp{Name: oname.Text, Pos: oname.Pos, Doc: p.docFor(oname.Pos.Line)}
		if p.at(LParen) {
			for p.at(LParen) {
				op.Binders = append(op.Binders, p.parseBinder())
			}
			p.expect(Colon, "`:` after operation binders")
			op.Result = p.parseType()
			if _, ok := p.accept(Assign); ok {
				op.Default = p.parseExpr()
			}
		} else {
			p.expect(Colon, "`:` in operation signature")
			op.Sig = p.parseType()
		}
		d.Ops = append(d.Ops, op)
	}
	return d
}

func (p *parser) parseInstance(doc []string, attrs []Attr) *InstanceDecl {
	name := p.expect(IDENT, "instance name")
	d := &InstanceDecl{Doc: doc, Attrs: attrs, Name: name.Text, Pos: name.Pos}
	for p.at(LParen) {
		d.Binders = append(d.Binders, p.parseBinder())
	}
	p.expect(Colon, "`:` before instance head")
	d.Head = p.parseTypeApp()
	p.expect(KwWhere, "`where`")
	for p.at(IDENT) {
		mname := p.expect(IDENT, "member name")
		m := &InstMember{Name: mname.Text, Pos: mname.Pos}
		if p.at(LParen) {
			// Typed form: Combine (a b : Int) : Int := ...
			for p.at(LParen) {
				m.Binders = append(m.Binders, p.parseBinder())
			}
			p.expect(Colon, "`:` after member binders")
			m.Result = p.parseType()
		} else {
			// Bare form: Combine a b := ... — types inferred from the
			// class operation (local classes only).
			for p.at(IDENT) {
				m.Bare = append(m.Bare, p.tok().Text)
				p.i++
			}
			if _, ok := p.accept(Colon); ok {
				m.Result = p.parseType()
			}
		}
		p.expect(Assign, "`:=` in instance member")
		m.Body = p.parseExpr()
		d.Members = append(d.Members, m)
	}
	return d
}

func (p *parser) parseNamespace() *NamespaceDecl {
	name := p.expect(IDENT, "namespace name")
	d := &NamespaceDecl{Name: name.Text, Pos: name.Pos}
	for !p.at(KwEnd) {
		if p.at(EOF) {
			p.fail(name.Pos, "namespace %s is missing `end`", d.Name)
		}
		d.Decls = append(d.Decls, p.parseDecl())
	}
	p.expect(KwEnd, "`end`")
	return d
}

// ----------------------------------------------------------- expressions

var binopPrec = map[Kind]int{
	Pipe: 1, Compose: 1, Kleisli: 1,
	OrOr:   2,
	AndAnd: 3,
	EqEq:   4, NotEq: 4, Lt: 4, LtEq: 4, Gt: 4, GtEq: 4,
	Plus: 5, Minus: 5,
	Star: 6, Slash: 6, Percent: 6,
}

var binopText = map[Kind]string{
	Pipe: "|>", Compose: ">>>", Kleisli: ">=>", OrOr: "||", AndAnd: "&&",
	EqEq: "==", NotEq: "!=", Lt: "<", LtEq: "<=", Gt: ">", GtEq: ">=",
	Plus: "+", Minus: "-", Star: "*", Slash: "/", Percent: "%",
}

func (p *parser) parseExpr() Expr {
	t := p.tok()
	switch t.Kind {
	case KwFun:
		p.i++
		f := &Fun{Pos: t.Pos}
		for p.at(LParen) || p.at(IDENT) {
			if name, ok := p.accept(IDENT); ok {
				// A bare binder; its type comes from the expected type at
				// the lambda's position.
				f.Binders = append(f.Binders, &Binder{Names: []string{name.Text}, Pos: name.Pos})
				continue
			}
			f.Binders = append(f.Binders, p.parseBinder())
		}
		if _, ok := p.accept(Colon); ok {
			f.Result = p.parseType()
		}
		p.expect(FatArrow, "`=>` in lambda")
		f.Body = p.parseExpr()
		return f
	case KwMatch:
		p.i++
		m := &Match{Pos: t.Pos}
		m.Subject = p.parseOp(1)
		p.expect(KwWith, "`with`")
		m.Clauses = p.parseClauses()
		return m
	case KwIf:
		p.i++
		e := &If{Pos: t.Pos}
		e.Cond = p.parseOp(1)
		p.expect(KwThen, "`then`")
		e.Then = p.parseExpr()
		p.expect(KwElse, "`else`")
		e.Else = p.parseExpr()
		return e
	case KwLet:
		p.i++
		e := &LetIn{Pos: t.Pos}
		e.Pat = p.parsePattern()
		if _, ok := p.accept(Colon); ok {
			e.Type = p.parseType()
		}
		p.expect(Assign, "`:=` in let")
		e.Val = p.parseExpr()
		p.expect(Semi, "`;` after let binding")
		e.Body = p.parseExpr()
		return e
	case KwLetStar:
		p.i++
		e := &LetStar{Pos: t.Pos}
		e.Pat = p.parsePattern()
		if _, ok := p.accept(Colon); ok {
			e.Type = p.parseType()
		}
		p.expect(Eq, "`=` in let*")
		e.Val = p.parseExpr()
		p.expect(KwIn, "`in` after let* binding")
		e.Body = p.parseExpr()
		return e
	case KwDo:
		p.i++
		return p.parseDoBlock(t.Pos)
	case KwSelect:
		p.i++
		return p.parseSelect(t.Pos)
	case KwWhile, KwFor, KwReturn, KwDefer, KwGo:
		p.fail(t.Pos, "%q is a do-block statement; wrap it in do { ... }", t.Text)
	}
	return p.parseOp(1)
}

// ------------------------------------------------------------- do blocks

func (p *parser) parseDoBlock(pos Pos) *DoBlock {
	p.expect(LBrace, "`{` after do")
	b := &DoBlock{Pos: pos}
	for !p.at(RBrace) {
		if p.at(EOF) {
			p.fail(pos, "do block is missing `}`")
		}
		b.Stmts = append(b.Stmts, p.parseDoStmt())
		p.accept(Semi)
	}
	p.expect(RBrace, "`}` closing do block")
	return b
}

// parseSimpleNames parses a comma-separated binder list of identifiers
// (including _) for let/for statements.
func (p *parser) parseSimpleNames(what string) []string {
	var names []string
	for {
		names = append(names, p.expect(IDENT, what).Text)
		if _, ok := p.accept(Comma); !ok {
			return names
		}
	}
}

func (p *parser) parseDoStmt() DoStmt {
	t := p.tok()
	switch t.Kind {
	case KwLet:
		p.i++
		s := &DoLet{Pos: t.Pos}
		if _, ok := p.accept(KwMut); ok {
			s.Mut = true
		}
		s.Names = p.parseSimpleNames("binder name")
		if _, ok := p.accept(Colon); ok {
			s.Type = p.parseType()
		}
		p.expect(Assign, "`:=` in let")
		s.Val = p.parseExpr()
		return s
	case KwWhile:
		p.i++
		s := &DoWhile{Pos: t.Pos}
		s.Cond = p.parseOp(1)
		p.expect(KwDo, "`do` after while condition")
		s.Body = p.parseDoBlock(t.Pos)
		return s
	case KwFor:
		p.i++
		s := &DoFor{Pos: t.Pos}
		s.Names = p.parseSimpleNames("range binder")
		p.expect(KwIn, "`in` after for binders")
		s.Seq = p.parseOp(1)
		p.expect(KwDo, "`do` after for range")
		s.Body = p.parseDoBlock(t.Pos)
		return s
	case KwDefer:
		p.i++
		return &DoDefer{Call: p.parseExpr(), Pos: t.Pos}
	case KwGo:
		p.i++
		return &DoGo{Call: p.parseExpr(), Pos: t.Pos}
	case KwReturn:
		p.i++
		if p.at(Semi) || p.at(RBrace) {
			return &DoReturn{Pos: t.Pos}
		}
		return &DoReturn{Val: p.parseExpr(), Pos: t.Pos}
	}
	e := p.parseExpr()
	if arrow, ok := p.accept(LArrow); ok {
		// A channel send: ch <- v.
		return &DoSend{Chan: e, Val: p.parseExpr(), Pos: arrow.Pos}
	}
	if _, ok := p.accept(Assign); ok {
		switch e.(type) {
		case *Ident, *Selector:
		default:
			p.fail(t.Pos, "assignment targets are locals or field paths")
		}
		return &DoAssign{Target: e, Val: p.parseExpr(), Pos: t.Pos}
	}
	return &DoExprStmt{X: e, Pos: t.Pos}
}

// --------------------------------------------------------------- select

func (p *parser) parseSelect(pos Pos) *SelectExpr {
	p.expect(KwWith, "`with` after select")
	sel := &SelectExpr{Pos: pos}
	for p.at(Bar) {
		bar := p.tok()
		p.i++
		arm := &SelectArm{Pos: bar.Pos}
		if _, ok := p.accept(KwDefault); ok {
			arm.Kind = "default"
		} else {
			pat := p.parsePattern()
			switch pat.(type) {
			case *PBind, *PWild:
			default:
				p.fail(bar.Pos, "select binds a plain name or _")
			}
			arm.Pat = pat
			p.expect(LArrow, "`<-` in select arm")
			switch {
			case p.at(KwRecv):
				p.i++
				arm.Kind = "recv"
				arm.Chan = p.parseOp(1)
			case p.at(KwSend):
				p.i++
				arm.Kind = "send"
				arm.Chan = p.parsePostfix()
				arm.Val = p.parseOp(1)
			default:
				p.fail(p.tok().Pos, "expected `recv` or `send` in select arm")
			}
		}
		p.expect(FatArrow, "`=>` in select arm")
		arm.Body = p.parseDoStmt()
		sel.Arms = append(sel.Arms, arm)
	}
	if len(sel.Arms) == 0 {
		p.fail(pos, "select needs at least one arm")
	}
	return sel
}

func (p *parser) parseOp(minPrec int) Expr {
	left := p.parseOperand()
	for {
		t := p.tok()
		prec, ok := binopPrec[t.Kind]
		if !ok || prec < minPrec {
			return left
		}
		p.i++
		right := p.parseOpRight(prec + 1)
		left = &Binop{Op: binopText[t.Kind], L: left, R: right, Pos: t.Pos}
	}
}

// parseOpRight admits keyword expressions on the right of a binop only
// when parenthesized (parseOperand handles parens); bare keyword
// expressions on an operator's right would be ambiguous.
func (p *parser) parseOpRight(minPrec int) Expr {
	return p.parseOp(minPrec)
}

func (p *parser) parseOperand() Expr {
	e := p.parseUnary()
	for {
		if _, ok := p.accept(Question); ok {
			e = &Try{X: e, Pos: e.exprPos()}
			continue
		}
		return e
	}
}

func (p *parser) parseUnary() Expr {
	if t, ok := p.accept(Minus); ok {
		return &Unary{Op: "-", X: p.parseUnary(), Pos: t.Pos}
	}
	if t, ok := p.accept(Bang); ok {
		return &Unary{Op: "!", X: p.parseUnary(), Pos: t.Pos}
	}
	if t, ok := p.accept(Amp); ok {
		return &Unary{Op: "&", X: p.parseUnary(), Pos: t.Pos}
	}
	if t, ok := p.accept(LArrow); ok {
		// A channel receive: <-ch.
		return &Unary{Op: "<-", X: p.parseUnary(), Pos: t.Pos}
	}
	if t, ok := p.accept(Star); ok {
		// Prefix `*` dereferences; binary multiplication is only ever
		// parsed after an operand, so the two cannot collide.
		return &Unary{Op: "*", X: p.parseUnary(), Pos: t.Pos}
	}
	return p.parseApp()
}

// parseArg parses one juxtaposition argument, admitting the prefix
// operators that Go APIs need (`f &x`). Multiplication is excluded:
// `f * x` must keep reading as a product.
func (p *parser) parseArg() Expr {
	if t, ok := p.accept(Amp); ok {
		return &Unary{Op: "&", X: p.parseArg(), Pos: t.Pos}
	}
	return p.parsePostfix()
}

func (p *parser) parseApp() Expr {
	head := p.parsePostfix()
	var args []Expr
	// Application arguments must start on the same line as the token
	// before them; parenthesize an application to span lines. Without
	// this rule juxtaposition would swallow the next declaration's name.
	for p.atOperandStart() && p.tok().Pos.Line == p.toks[p.i-1].Pos.Line {
		args = append(args, p.parseArg())
	}
	if len(args) == 0 {
		return head
	}
	return &App{Fn: head, Args: args, Pos: head.exprPos()}
}

func (p *parser) atOperandStart() bool {
	switch p.tok().Kind {
	case IDENT, INT, FLOAT, STRING, LParen, At, Amp:
		return true
	case LBrack:
		// A spaced `[` opens a list-literal argument; an adjacent one
		// indexes the operand before it and never reaches here.
		return true
	case Question:
		return p.atHole()
	}
	return false
}

// atHole reports whether the current `?` opens a typed hole rather than a
// postfix try. A hole's name is attached to its `?` and the `?` is detached
// from whatever precedes it, so `f ?x` passes a hole while `x? f` tries.
func (p *parser) atHole() bool {
	q := p.tok()
	if q.Kind != Question {
		return false
	}
	name := p.toks[p.i+1]
	if name.Kind != IDENT || name.Pos.Line != q.Pos.Line || name.Pos.Col != q.Pos.Col+1 {
		return false
	}
	if p.i > 0 {
		prev := p.toks[p.i-1]
		if prev.Pos.Line == q.Pos.Line && prev.Pos.Col+utf8.RuneCountInString(prev.Text) == q.Pos.Col {
			return false // glued to the operand before it: a postfix try
		}
	}
	return true
}

// parseHole consumes `?name`.
func (p *parser) parseHole() Expr {
	q := p.tok()
	p.i++
	name := p.expect(IDENT, "a hole name")
	return &Hole{Name: name.Text, Pos: q.Pos}
}

func (p *parser) parsePostfix() Expr {
	e := p.parseAtom()
	for {
		if p.at(Dot) && p.toks[p.i+1].Kind == IDENT {
			p.i++
			name := p.expect(IDENT, "selector")
			e = &Selector{X: e, Name: name.Text, Pos: e.exprPos()}
			continue
		}
		// A record literal: Config { Port = 8080, Name = "x" }. Claimed
		// only after a Capitalized name, the one place `{` can follow an
		// expression (do blocks are introduced by `do`).
		if p.at(LBrace) && isRecordHead(e) {
			e = p.parseRecordLit(e)
			continue
		}
		// An index: xs[i], with the `[` glued to the expression. A spaced
		// `[` is a list-literal argument instead (f [1, 2]).
		if p.at(LBrack) && p.tok().Adj {
			lb := p.tok()
			p.i++
			idx := p.parseExpr()
			p.expect(RBrack, "`]` closing an index")
			e = &IndexExpr{X: e, Index: idx, Pos: lb.Pos}
			continue
		}
		return e
	}
}

// isRecordHead reports whether e can head a record literal: a
// Capitalized name, optionally qualified (pkg.Name).
func isRecordHead(e Expr) bool {
	switch e := e.(type) {
	case *Ident:
		return isUpperName(e.Name)
	case *Selector:
		if id, ok := e.X.(*Ident); ok {
			return !isUpperName(id.Name) && isUpperName(e.Name)
		}
	}
	return false
}

func (p *parser) parseRecordLit(head Expr) *RecordLit {
	lb := p.expect(LBrace, "`{`")
	lit := &RecordLit{Type: head, Pos: lb.Pos}
	for !p.at(RBrace) {
		name := p.expect(IDENT, "field name")
		p.expect(Eq, "`=` in record literal")
		lit.Fields = append(lit.Fields, &FieldVal{Name: name.Text, Val: p.parseExpr(), Pos: name.Pos})
		if _, ok := p.accept(Comma); !ok {
			break
		}
	}
	p.expect(RBrace, "`}` closing record literal")
	return lit
}

func (p *parser) parseAtom() Expr {
	t := p.tok()
	switch t.Kind {
	case IDENT:
		p.i++
		return &Ident{Name: t.Text, Pos: t.Pos}
	case INT, FLOAT, STRING:
		p.i++
		return &Lit{Kind: t.Kind, Text: t.Text, Pos: t.Pos}
	case LParen:
		p.i++
		if _, ok := p.accept(RParen); ok {
			return &Unit{Pos: t.Pos}
		}
		inner := p.parseExpr()
		p.expect(RParen, "`)` closing expression")
		return inner
	case LBrack:
		p.i++
		lit := &ListLit{Pos: t.Pos}
		for !p.at(RBrack) {
			lit.Elems = append(lit.Elems, p.parseExpr())
			if _, ok := p.accept(Comma); !ok {
				break
			}
		}
		p.expect(RBrack, "`]` closing list literal")
		return lit
	case LBrace:
		p.i++
		base := p.parseExpr()
		p.expect(KwWith, "`with` in a record update ({ r with Field = v })")
		u := &RecordUpdate{Base: base, Pos: t.Pos}
		for !p.at(RBrace) {
			name := p.expect(IDENT, "field name")
			p.expect(Eq, "`=` in record update")
			u.Fields = append(u.Fields, &FieldVal{Name: name.Text, Val: p.parseExpr(), Pos: name.Pos})
			if _, ok := p.accept(Comma); !ok {
				break
			}
		}
		if len(u.Fields) == 0 {
			p.fail(t.Pos, "a record update names at least one field")
		}
		p.expect(RBrace, "`}` closing record update")
		return u
	case At:
		p.i++
		name := p.expect(IDENT, "instance name after `@`")
		return &Witness{Name: name.Text, Pos: t.Pos}
	case Dot:
		p.i++
		name := p.expect(IDENT, "member name after `.`")
		return &DotSegment{Name: name.Text, Pos: t.Pos}
	case Question:
		if p.atHole() {
			return p.parseHole()
		}
		p.fail(t.Pos, "a typed hole is spelled ?name, with the name attached to the `?`")
	}
	p.fail(t.Pos, "expected an expression, found %q", t.Text)
	return nil
}
