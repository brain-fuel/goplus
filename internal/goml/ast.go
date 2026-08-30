package goml

// File is a parsed .goml source file.
type File struct {
	Attrs   []Attr // module-level attributes, e.g. @[laws "out=lawtest"]
	Module  string
	Imports []*Import
	Opens   []*Open
	Decls   []Decl
}

// Open is `open pkg exposing (A, B)`: the named Capitalized members of
// an imported package become usable unqualified. Names are Capitalized
// by rule, so an opened name can never collide with a binder.
type Open struct {
	Pkg   string
	Names []string
	Pos   Pos
}

// Import is one import declaration.
type Import struct {
	Path  string // quoted string as written
	Alias string // "" without `as`
}

// Attr is one @[...] attribute item: a head identifier plus raw
// argument tokens (identifiers, strings, numbers) as written.
type Attr struct {
	Name string
	Args []string
	Pos  Pos
}

// Decl is a top-level declaration.
type Decl interface{ declPos() Pos }

// Binder is one binder group: (x y : T), {0 n : Nat}, [Monoid t], or the
// unit binder () that makes a declaration a nullary function.
type Binder struct {
	Implicit bool   // { ... }
	Instance bool   // [ ... ] — class constraint; Type holds the class app
	Unit     bool   // () — takes no parameters; marks a function, not a value
	Quantity string // "", "0", "1", or a multiplicity variable name
	Names    []string
	Type     Type
	Pos      Pos
}

// Clause is one `| patterns => expr` alternative. Alts holds
// or-pattern alternatives (single column); Row holds a comma-separated
// multi-column pattern row (clausal definitions over several
// arguments). The two forms do not mix.
type Clause struct {
	Alts []Pattern // or-pattern alternatives (usually one)
	Row  []Pattern // multi-column row; nil for single-column clauses
	Body Expr
	Pos  Pos
}

// LetDecl is a top-level (or namespace-level) function/value binding.
type LetDecl struct {
	Doc     []string
	Attrs   []Attr
	Total   bool
	Rec     bool
	Name    string
	Binders []*Binder
	Result  Type // result annotation (":= form"); nil when omitted
	Sig     Type // clausal form: the full signature type
	Body    Expr // ":= form" body; nil for clausal form
	Clauses []*Clause
	Where   []*LetDecl // where-helpers: closed, package-private lets
	Pos     Pos
}

func (d *LetDecl) declPos() Pos { return d.Pos }

// Field is a record field or a constructor field.
type Field struct {
	Name  string
	Type  Type
	Attrs []Attr
	Pos   Pos
}

// Ctor is one sum-type constructor.
type Ctor struct {
	Doc    []string
	Name   string
	Exist  []*Binder // existential binders: {a : fmt.Stringer}
	Fields []*Field
	Result Type // pinned result (GADT / where-form); nil otherwise
	Pos    Pos
}

// TypeDecl is a type declaration in any of its forms.
type TypeDecl struct {
	Doc      []string
	Attrs    []Attr
	Name     string
	Binders  []*Binder
	Kind     []Type // index sorts from `: S1 -> S2 -> Type`
	Sum      []*Ctor
	Where    bool // GADT where-form
	Record   []*Field
	Refine   *Refine
	Prop     Type       // named proposition body: := prop { P }
	Iface    *IfaceBody // interface body: := interface { … }
	Alias    Type
	Deriving []string
	Pos      Pos
}

// IfaceBody is an interface type body.
type IfaceBody struct {
	Members []*IfaceMember
}

// IfaceMember is one interface member: a method signature (curried
// arrow type, flattened to a Go method) or an embedded interface.
type IfaceMember struct {
	Doc  []string
	Name string // method name; "" for an embedded interface
	Sig  Type   // method arrow type, or the embedded type
	Pos  Pos
}

func (d *TypeDecl) declPos() Pos { return d.Pos }

// Refine is a refinement body { v : Base | pred }.
type Refine struct {
	Binder string
	Base   Type
	Pred   Expr
}

// ClassOp is a class operation, in signature or binder form.
type ClassOp struct {
	Doc     []string
	Name    string
	Binders []*Binder // binder form
	Result  Type      // binder form result
	Sig     Type      // signature form: full arrow type
	Default Expr      // optional default body (binder form only)
	Pos     Pos
}

// Law is a class law.
type Law struct {
	Doc     []string
	Name    string
	Binders []*Binder
	Body    Expr
	Pos     Pos
}

// ClassDecl is a typeclass declaration.
type ClassDecl struct {
	Doc     []string
	Attrs   []Attr
	Name    string
	Binder  *Binder // exactly one type binder in v0
	Extends []Type
	Ops     []*ClassOp
	Laws    []*Law
	Pos     Pos
}

func (d *ClassDecl) declPos() Pos { return d.Pos }

// InstMember is one instance member definition.
type InstMember struct {
	Name    string
	Binders []*Binder // typed form
	Bare    []string  // bare form: names typed by the class operation
	Result  Type
	Body    Expr
	Pos     Pos
}

// InstanceDecl is a (named) class instance.
type InstanceDecl struct {
	Doc     []string
	Attrs   []Attr
	Name    string
	Binders []*Binder // generic-instance binders
	Head    Type      // fully applied class: Group Int
	Members []*InstMember
	Pos     Pos
}

func (d *InstanceDecl) declPos() Pos { return d.Pos }

// NamespaceDecl groups declarations under a type's namespace; lets whose
// first binder has the namespace type become methods.
type NamespaceDecl struct {
	Name  string
	Decls []Decl
	Pos   Pos
}

func (d *NamespaceDecl) declPos() Pos { return d.Pos }

// ---------------------------------------------------------------- types

// Type is a goml type expression.
type Type interface{ typePos() Pos }

// TypeName is a possibly qualified type name (or type variable).
type TypeName struct {
	Pkg  string // "" unqualified
	Name string
	Pos  Pos
}

func (t *TypeName) typePos() Pos { return t.Pos }

// TypeApp is application of a head name to arguments: Vec a (n + 1).
type TypeApp struct {
	Head *TypeName
	Args []Type
	Pos  Pos
}

func (t *TypeApp) typePos() Pos { return t.Pos }

// TypeArrow is a function type (right associative).
type TypeArrow struct {
	From, To Type
	Pos      Pos
}

func (t *TypeArrow) typePos() Pos { return t.Pos }

// TypeNat is a natural-number literal in type/index position.
type TypeNat struct {
	Lit string
	Pos Pos
}

func (t *TypeNat) typePos() Pos { return t.Pos }

// TypeIndexOp is index arithmetic: n + 1, n + m.
type TypeIndexOp struct {
	Op   string
	L, R Type
	Pos  Pos
}

func (t *TypeIndexOp) typePos() Pos { return t.Pos }

// TypeEq is the propositional-equality type n = m (lowered to Eq[n, m]).
type TypeEq struct {
	L, R Type
	Pos  Pos
}

func (t *TypeEq) typePos() Pos { return t.Pos }

// ---------------------------------------------------------------- exprs

// Expr is a goml expression.
type Expr interface{ exprPos() Pos }

// Ident is a bare identifier reference.
type Ident struct {
	Name string
	Pos  Pos
}

func (e *Ident) exprPos() Pos { return e.Pos }

// Lit is an int/float/string literal.
type Lit struct {
	Kind Kind // INT, FLOAT, STRING
	Text string
	Pos  Pos
}

func (e *Lit) exprPos() Pos { return e.Pos }

// Unit is the () expression.
type Unit struct{ Pos Pos }

func (e *Unit) exprPos() Pos { return e.Pos }

// Selector is field/method selection: x.name (possibly chained).
type Selector struct {
	X    Expr
	Name string
	Pos  Pos
}

func (e *Selector) exprPos() Pos { return e.Pos }

// DotSegment is a leading-dot pipeline segment: .UnwrapOr 0.
type DotSegment struct {
	Name string
	Args []Expr
	Pos  Pos
}

func (e *DotSegment) exprPos() Pos { return e.Pos }

// IndexExpr is a slice, array, or map index: xs[i].
type IndexExpr struct {
	X     Expr
	Index Expr
	Pos   Pos
}

func (e *IndexExpr) exprPos() Pos { return e.Pos }

// Witness is an explicit instance witness argument: @IntAdd.
type Witness struct {
	Name string
	Pos  Pos
}

func (e *Witness) exprPos() Pos { return e.Pos }

// App is juxtaposition application, flattened: f x y.
type App struct {
	Fn   Expr
	Args []Expr
	Pos  Pos
}

func (e *App) exprPos() Pos { return e.Pos }

// Binop is a binary operation (incl. |>, >>>, >=>).
type Binop struct {
	Op   string
	L, R Expr
	Pos  Pos
}

func (e *Binop) exprPos() Pos { return e.Pos }

// Unary is unary negation.
type Unary struct {
	Op  string
	X   Expr
	Pos Pos
}

func (e *Unary) exprPos() Pos { return e.Pos }

// Try is postfix ?.
type Try struct {
	X   Expr
	Pos Pos
}

func (e *Try) exprPos() Pos { return e.Pos }

// Hole is a typed hole: `?name`, standing where code is not written yet.
// The name travels verbatim into the lowered .gp text, where the core
// reports the hole's goal.
type Hole struct {
	Name string
	Pos  Pos // position of the '?'
}

func (e *Hole) exprPos() Pos { return e.Pos }

// If is if/then/else (always both branches).
type If struct {
	Cond, Then, Else Expr
	Pos              Pos
}

func (e *If) exprPos() Pos { return e.Pos }

// Match is match ... with clauses.
type Match struct {
	Subject Expr
	Clauses []*Clause
	Pos     Pos
}

func (e *Match) exprPos() Pos { return e.Pos }

// LetIn is `let pat := e; body`.
type LetIn struct {
	Pat  Pattern
	Type Type // optional annotation
	Val  Expr
	Body Expr
	Pos  Pos
}

func (e *LetIn) exprPos() Pos { return e.Pos }

// Fun is an annotated lambda.
type Fun struct {
	Binders []*Binder
	Result  Type
	Body    Expr
	Pos     Pos
}

func (e *Fun) exprPos() Pos { return e.Pos }

// ------------------------------------------------------------- patterns

// Pattern is a match pattern.
type Pattern interface{ patPos() Pos }

// PWild is _.
type PWild struct{ Pos Pos }

func (p *PWild) patPos() Pos { return p.Pos }

// PBind binds the scrutinee (lowercase identifier).
type PBind struct {
	Name string
	Pos  Pos
}

func (p *PBind) patPos() Pos { return p.Pos }

// PCtor is a constructor pattern, optionally with `as` binding.
type PCtor struct {
	Pkg  string
	Name string
	Args []Pattern
	As   string
	Pos  Pos
}

func (p *PCtor) patPos() Pos { return p.Pos }

// LetStar is monadic bind: `let* pat = e in body` (lowers to `pat := e?`).
type LetStar struct {
	Pat  Pattern
	Type Type
	Val  Expr
	Body Expr
	Pos  Pos
}

func (e *LetStar) exprPos() Pos { return e.Pos }

// DoBlock is the Go-statement embedding: do { stmt; ... }.
type DoBlock struct {
	Stmts []DoStmt
	Pos   Pos
}

func (e *DoBlock) exprPos() Pos { return e.Pos }

// DoStmt is one statement inside a do block.
type DoStmt interface{ stmtPos() Pos }

// DoLet declares (optionally mutable) locals: let mut a, b := e.
type DoLet struct {
	Mut   bool
	Names []string // "_" allowed
	Type  Type
	Val   Expr
	Pos   Pos
}

func (s *DoLet) stmtPos() Pos { return s.Pos }

// DoAssign assigns to a declared local or field path: x := e, b.left := e.
type DoAssign struct {
	Target Expr // Ident or Selector chain
	Val    Expr
	Pos    Pos
}

func (s *DoAssign) stmtPos() Pos { return s.Pos }

// DoWhile is `while cond do { ... }`.
type DoWhile struct {
	Cond Expr
	Body *DoBlock
	Pos  Pos
}

func (s *DoWhile) stmtPos() Pos { return s.Pos }

// DoFor is `for a, b in e do { ... }` (Go range).
type DoFor struct {
	Names []string
	Seq   Expr
	Body  *DoBlock
	Pos   Pos
}

func (s *DoFor) stmtPos() Pos { return s.Pos }

// DoSend is a channel send: ch <- v.
type DoSend struct {
	Chan Expr
	Val  Expr
	Pos  Pos
}

func (s *DoSend) stmtPos() Pos { return s.Pos }

// DoDefer and DoGo wrap a call.
type DoDefer struct {
	Call Expr
	Pos  Pos
}

func (s *DoDefer) stmtPos() Pos { return s.Pos }

// DoGo launches a goroutine.
type DoGo struct {
	Call Expr
	Pos  Pos
}

func (s *DoGo) stmtPos() Pos { return s.Pos }

// DoReturn is `return [e]`.
type DoReturn struct {
	Val Expr // nil for bare return
	Pos Pos
}

func (s *DoReturn) stmtPos() Pos { return s.Pos }

// DoExprStmt is a bare expression statement (calls, select, unit).
type DoExprStmt struct {
	X   Expr
	Pos Pos
}

func (s *DoExprStmt) stmtPos() Pos { return s.Pos }

// SelectExpr is `select with | arm ...`, lowered to a native Go select.
type SelectExpr struct {
	Arms []*SelectArm
	Pos  Pos
}

func (e *SelectExpr) exprPos() Pos { return e.Pos }

// SelectArm is one communication clause.
type SelectArm struct {
	Kind string  // "recv", "send", "default"
	Pat  Pattern // recv: PBind or PWild
	Chan Expr
	Val  Expr // send only
	Body DoStmt
	Pos  Pos
}

// RecordLit is a record (Go struct) literal: Config { Port = 8080 }.
type RecordLit struct {
	Type   Expr // Ident or pkg-qualified Selector naming the type
	Fields []*FieldVal
	Pos    Pos
}

func (e *RecordLit) exprPos() Pos { return e.Pos }

// FieldVal is one `Name = expr` element of a record literal.
type FieldVal struct {
	Name string
	Val  Expr
	Pos  Pos
}

// Impossible is the `impossible` arm body: an assertion that the arm's
// pattern is ruled out by the scrutinee's indices, checked and dropped.
type Impossible struct{ Pos Pos }

func (e *Impossible) exprPos() Pos { return e.Pos }

// ListLit is a list literal [e1, e2], lowered to a Go slice literal.
// Its element type comes from the expected type at its position.
type ListLit struct {
	Elems []Expr
	Pos   Pos
}

func (e *ListLit) exprPos() Pos { return e.Pos }

// RecordUpdate is a functional field update: { r with Port = p }. It
// lowers to a hoisted copy-then-assign, so the base is untouched.
type RecordUpdate struct {
	Base   Expr
	Fields []*FieldVal
	Pos    Pos
}

func (e *RecordUpdate) exprPos() Pos { return e.Pos }
