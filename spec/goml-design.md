# goml — an ML-family surface for the Go+ core (design proposal)

Status: **v1 front end implemented and released** (goplus v0.145.0): the `goml` facade package, the `cmd/goml` binary, and
pipeline support for `.goml` sources — see §10 for the surface v1
covers and the deliberate deferrals.

Original status: proposal / not implemented. This document suggests a second
surface syntax — `goml`, in the SML/OCaml/Idris2/Lean4 family — for the
existing Go+ semantic core, a grammar with examples, a feature-parity map
against `.gp`, and the idioms that become materially cleaner when freed
from the strict-Go-superset constraint.

## 1. Motivation and positioning

Go+ (`.gp`) is deliberately a **strict superset of Go**: every extension is
smuggled through spellings that are invalid Go (`0 n nat`, `type T class {`,
`instance Name`, contextual `total`/`tail`/`refine`/`delegate`). That
constraint bought seamless adoption and diff-sized grammar deltas, but it
taxes exactly the programs the dependent core was built for: erased indices
must be spelled in every signature, constructors need explicit
instantiation (`Err[U, E](e)`), unused binders need `_ = t`, statements
dominate where expressions want to be, and eight keywords exist only as
parser contortions.

goml is **not a new language**. It is a second front end over the same
elaborated core — the QTT quantities, indexed enums, classes/instances/laws,
refinements, totality, and linearity that `.gp` already checks — lowering
through the same pipeline to the same idiomatic, committed Go
(`*_gml.go` beside `*_gp.go`, same `//goplus:` markers). One core, two
surfaces, one output:

```
 .gp  ──parse──▸ surface AST ─┐
                              ├─▸ shared core (resolve, dependent check,
 .goml ─parse──▸ surface AST ─┘    decider, class resolution, refinement
                                   proof) ──lower──▸ idiomatic Go (+ Java)
```

Parity is therefore **by construction** for everything type-system-shaped;
the design work is (a) an ML grammar whose desugarings land on existing core
nodes, and (b) an honest story for the Go-statement material (`defer`, `go`,
channels, mutation) that `.gp` gets for free.

## 2. Design principles

1. **Same core, second syntax.** Every goml construct desugars to a core
   node `.gp` can already produce; anything requiring a new core feature is
   out of scope for parity v1 (flagged below).
2. **Not a Go superset — so take the wins.** Real keywords, expression
   orientation, currying, inference, implicit binders, or-patterns,
   attributes instead of directive comments.
3. **Go-facing output is unchanged.** Generated packages remain plain Go a
   Go consumer calls without knowing which surface authored them. Mixed
   packages (`.gp` and `.goml` files side by side) are legal; markers make
   cross-surface, cross-package features work identically.
4. **Go's export rule, verbatim** *(decided 2026-08-11)*. A Capitalized
   declaration is exported, a lowercase one is package-private — exactly
   Go and `.gp`. Go names are identical to goml names: no case mapping,
   no collision rule, and generated docs match the source spelling. In
   patterns a lowercase identifier is a binder and a Capitalized one is a
   constructor (the ML rule, which Go's convention happens to satisfy);
   primes and other non-Go identifier characters are excluded since names
   travel verbatim into Go.

## 3. Lexical structure and modules

- Files are `.goml`; `goplus gen` (and therefore `go generate ./...`)
  produces `<file>_gml.go` beside the source, protobuf-style, committed.
- A file opens with `module <name>` (the Go package name), then imports:

```
module vec

import "goforge.dev/goplus/std/result" as result
import "fmt"                        -- Go packages import directly
open result                          -- optional unqualified access
```

- Comments: `-- line` and `(* block *)`. Layout is significant the Lean
  way (a `where`/`with` block continues while indented); braces and `;`
  are always accepted as the explicit form, so generated/one-line code
  needs no layout.
- Predeclared type spellings map onto Go: `Int, Int64, UInt64, Float64,
  Bool, String, Byte, Rune, Unit, Nat, Mult, Error`; type formers
  `Ptr t`, `Slice t`, `Array n t`, `Map k v`, `Chan t`, `Func`, tuples
  `(a, b)`. Qualified Go names (`os.File`, `fmt.Stringer`) are types as-is.

## 4. Grammar (EBNF sketch)

Deltas from this sketch would version exactly like `spec/grammar-*.ebnf`.

```ebnf
SourceFile   = "module" identifier { ImportDecl } { TopDecl } .
ImportDecl   = "import" string_lit [ "as" identifier ] | "open" identifier .

TopDecl      = { Attr } ( LetDecl | TypeDecl | ClassDecl | InstanceDecl
                        | NamespaceDecl ) .
Attr         = "@[" AttrItem { "," AttrItem } "]" .
AttrItem     = identifier { identifier | literal } .
             (* @[tail] @[box pointer] @[repr transparent] @[derive off]
                @[laws off] @[laws Int, String] @[gen Name] *)

LetDecl      = [ "total" ] "let" [ "rec" ] identifier { Binder }
               [ ":" Type ] ":=" Expr
             | [ "total" ] "let" [ "rec" ] identifier ":" Type Clauses .
Clauses      = { "|" PatternList [ Guard ] "=>" Expr } .        (* clausal defs *)
Guard        = "if" Expr .                                       (* reserved *)
Binder       = identifier
             | "(" [ Quantity ] IdentList ":" Type ")"           (* explicit *)
             | "{" [ Quantity ] IdentList ":" Type "}"           (* implicit *)
             | "[" Type "]" .                                    (* instance/class *)
Quantity     = "0" | "1" | identifier .                          (* QTT; ident names a Mult var *)

TypeDecl     = "type" identifier { Binder } [ ":" IndexArrow "Type" ]
               ( ":=" SumBody | "where" CtorSigs | ":=" RecordBody
               | ":=" RefineBody | ":=" PropBody | ":=" IfaceBody
               | ":=" Type )
               [ "deriving" identifier { "," identifier } ] .
IndexArrow   = { AtomicType "->" } .                             (* Nat -> State -> *)
SumBody      = [ "|" ] Ctor { "|" Ctor } .
Ctor         = identifier { CtorBinder } .
CtorBinder   = "(" identifier ":" Type ")" | "{" IdentList ":" Type "}" .
             (* braces: bounded existential fields *)
CtorSigs     = { "|" identifier { CtorBinder } ":" Type } .      (* GADT: pinned result *)
RecordBody   = "{" Field { ";" Field } "}" .
Field        = identifier ":" Type { Attr } .
             (* @[delegate]; @[json "port", yaml "port"] lowers to Go
                struct tags `json:"port" yaml:"port"` *)
RefineBody   = "{" identifier ":" Type "|" Expr "}" .            (* subset type *)
PropBody     = "prop" "{" Type "}" .                             (* named proposition *)
IfaceBody    = "interface" "{" [ IfaceMember { ";" IfaceMember } ] "}" .
IfaceMember  = identifier ":" Type | QualIdent .                 (* method / embedding *)

ClassDecl    = "class" identifier Binder [ "extends" Type { "," Type } ]
               "where" { ClassMember } .
ClassMember  = identifier ":" Type [ ":=" Expr ]                 (* op; body = default *)
             | "law" identifier { Binder } ":=" Expr .
InstanceDecl = "instance" [ identifier ":" ] Type
               "where" { identifier { Binder } ":=" Expr } .     (* named or anonymous *)
NamespaceDecl= "namespace" identifier { TopDecl } "end" .        (* Stack.map ⇒ method *)

Type         = AppType [ "->" Type ] | Binder "->" Type          (* Pi *)
             | AppType "=" AppType .                             (* propositional Eq *)
AppType      = AtomicType { AtomicType | IndexTerm } .
AtomicType   = QualIdent | "(" Type ")" | "(" Type { "," Type } ")" | nat_lit .
IndexTerm    = nat_lit | identifier | QualIdent { AtomicIndex }
             | IndexTerm ( "+" | "-" | "*" ) IndexTerm | TotalCall .

Expr         = "fun" { Binder } "=>" Expr
             | "match" Expr { "," Expr } "with" Clauses
             | "if" Expr "then" Expr "else" Expr
             | "let" Pattern [ ":" Type ] ":=" Expr ";" Expr     (* let-in *)
             | "let*" Pattern [ ":" Type ] "=" Expr "in" Expr    (* monadic bind *)
             | SelectExpr | DoExpr | OpExpr .
SelectExpr   = "select" "with" { "|" CommClause } .
CommClause   = Pattern "<-" "recv" Expr "=>" Expr
             | "_" "<-" "send" Expr Expr "=>" Expr
             | "default" "=>" Expr .
OpExpr       = (* precedence, loosest→tightest, mirroring .gp where shared:
                  |>  >>>  >=>   (loosest, left-assoc)
                  ||  &&  ==/!=/</<=/>/>=  +/-  * / %  
                  application by juxtaposition   f x y
                  @Witness explicit-instance application
                  postfix ?      .selector       (tightest) *)
DoExpr       = "do" "{" { DoStmt } "}" .
DoStmt       = "let" [ "mut" ] Pattern ":=" Expr
             | identifier ":=" Expr                              (* assign mut *)
             | "while" Expr DoBlock | "for" Pattern "in" Expr DoBlock
             | "defer" Expr | "go" Expr | "return" [ Expr ] | Expr .

Pattern      = "_" | identifier                                  (* lowercase: binder *)
             | QualCtor { AtomicPattern }                        (* Capitalized: ctor *)
             | "(" Pattern { "," Pattern } ")" | literal .       (* literals reserved *)
PatternList  = Pattern { "|" Pattern } .                         (* or-patterns *)
```

Notes:

- **Implicit binders `{n : Nat}`** are auto-generalized when unbound (an
  Idris-style free lowercase type/index variable in a signature is
  implicitly quantified), and solved by unification/the decider at call
  sites — they are exactly `.gp`'s 0-quantity inferred indices.
- **Instance binders `[Monoid t]`** are class constraints; they lower to
  the same leading witness parameters. `@IntAdd` applies a named witness
  explicitly (the `.gp` escape hatch `Accumulate(IntAdd, xs)`).
- **Or-patterns** subsume `.gp` v0.14 multi-pattern arms and go further:
  alternatives may bind, provided every alternative binds the same names
  at the same types; lowering duplicates the arm, so the core is untouched.
- **`let* pat = e in body`** desugars to `Bind e (fun pat => body)`,
  resolved method-aware against the bound expression's type exactly like a
  pipe segment (`Result.Bind` today; a `Monad` class instance once the
  stdlib earns one) — core-expressible, no new semantics. It coexists with
  postfix `?` *(decided)*: `let*` is the structured spelling for pipeline
  code, `?` the terse spelling for Go-boundary calls in `do` blocks.
- **`select`** is a match-like expression over channel operations, lowered
  to a native Go `select` (each `recv` arm binds the received value; a
  `default` arm makes it non-blocking) *(decided: ships in v1)*.
- **Guards and literal patterns** are reserved productions *(decided:
  core-first)*: goml v1 parses them behind a flag but rejects with a
  pointer to the shared-core milestone that lands them — exhaustiveness
  over literals needs decider work — after which `.gp` gains
  `case Cons(h, t) if h > n:` and goml gains `| Cons h t if h > n =>`
  simultaneously, preserving parity by construction.

## 5. Examples

Each example is chosen to mirror a shipped `.gp` milestone.

### 5.1 Sums, match, inference (v0.2 / v0.4 / v0.6)

```
module option

type Option (a : Type) :=
  | Some (value : a)
  | None

let Map (f : a -> b) : Option a -> Option b
  | Some v => Some (f v)          -- no [U] instantiation: inferred
  | None   => None

let UnwrapOr (o : Option a) (d : a) : a :=
  match o with
  | Some v => v
  | None   => d
```

Exhaustiveness, Maranget usefulness, constructor inference, and the
derived Go-facing `Fold`/`Cases` are unchanged — the last is *output-only*
in goml: the surface never needs a cases-struct because `match` is an
expression and functions are first-class.

### 5.2 Railway (v0.4)

```
-- do-block + postfix ?: terse at the Go boundary
let ReadPort (path : String) : Result Int Error := do {
  let raw := os.ReadFile path ?;       -- ? on Go-shaped (value, error)
  let n   := raw |> bytes.TrimSpace |> strconv.Atoi ?;
  Ok n
}

-- let*: the structured monadic spelling (desugars to Bind)
let Build (path : String) : Result App Error :=
  let* cfg = ReadPort path in
  let* db  = connect cfg in
  Ok (App cfg db)

-- track-aware pipeline, exactly .gp's |> lifting rules:
let n := s |> validate |> strings.TrimSpace |> strconv.Atoi |> .UnwrapOr 0

-- Kleisli:
let pipeline := strings.TrimSpace >=> validate >=> strconv.Atoi >=> save
```

### 5.3 The dependent core (v0.7) — `std/vec` reauthored

```
module vec

type Vec (a : Type) : Nat -> Type where
  | Nil                                    : Vec a 0
  | Cons (head : a) (tail : Vec a n)       : Vec a (n + 1)

let First : Vec a (n + 1) -> a             -- {0 n : Nat} auto-quantified
  | Cons h _ => h                          -- Nil impossible; wildcard, no `_ = t`

let Rest : Vec a (n + 1) -> Vec a n
  | Cons _ t => t

total let Concat (xs : Vec a n) (ys : Vec a m) : Vec a (n + m) :=
  match xs with
  | Nil       => ys
  | Cons h t  => Cons h (Concat t ys)

type Fin : Nat -> Type where
  | Zero              : Fin (n + 1)
  | Succ (prev : Fin n) : Fin (n + 1)

total let At (i : Fin n) (v : Vec a n) : a :=
  match i with
  | Zero   => First v
  | Succ p => At p (Rest v)

let Cast {0 n m : Nat} (0 p : n = m) (v : Vec a n) : Vec a m := v
let w := Cast refl v                       -- decider discharges 1+1 = 2
```

Compare `.gp` today: `func First[T any](0 n nat, v Vec[T, n+1]) T` with a
mandatory `_ = t` inside the arm. Same core obligations, same erasure, same
generated Go with runtime boundary guards.

### 5.4 QTT and linearity (v0.7)

```
let Process (1 f : Ptr os.File) : Error := f.Close ()   -- linear: exactly once
let DiscardProof {0 n : Nat} (v : Vec a n) : Int := Length v
let Dup {m : Mult} (m x : a) : (a, a) := (x, x)         -- multiplicity-poly (ill-typed at m=1)
```

`(0 x : T)` / `(1 x : T)` is Idris2's concrete syntax; `.gp`'s
`0 n nat` was always this notation squeezed into a Go parameter list. The
`m x ([]T)` parenthesization wart disappears — binder types are delimited.

### 5.5 Classes, named instances, laws (v0.5) — `std/algebra` reauthored

```
class Magma (t : Type) where
  Combine : t -> t -> t

class UnitalMagma (t : Type) extends Magma t where
  Empty : t
  law LeftId  (a : t) := Combine Empty a == a     -- == is derived Eq, not reflect
  law RightId (a : t) := Combine a Empty == a

instance IntAdd : Group Int where
  Combine a b := a + b
  Empty       := 0
  Invert a    := 0 - a

let Accumulate [Monoid t] (xs : Slice t) : t :=
  xs |> Fold Combine Empty

Accumulate @IntAdd [2, 3, 4]      -- 9  : explicit witness = naming the structure
Accumulate @IntMul [2, 3, 4]      -- 24
```

Named instances, subsumption (a `Group` witness answers a `Monoid`
constraint), defaults, ambiguity-as-error, and default-generated rapid law
tests are all core behavior. Only the law bodies improve: `==` resolves
through the derived structural-equality class instead of
`reflect.DeepEqual`, and lowers to the derived `TmEqual`-style functions.

### 5.6 Refinements (v0.22)

```
type Port     := { value : Int | 0 < value && value < 65536 }
type Positive := { value : Int | value > 0 }

let Listen (p : Port) : Error := net.ListenTCP p.value
let p := Port 8080                -- proved from the literal, or compile error
```

Set-builder comprehension is the notation refinement types have always
had on paper; `refine(value int) { … }` was its Go-shaped encoding.

### 5.7 Tail recursion (v0.21) — `recur` disappears

```
@[tail]
let rec SumTo (n acc : UInt64) : UInt64 :=
  if n == 0 then acc else SumTo (n - 1) (acc + n)
```

`@[tail]` demands every recursive call be in tail position (compile error
otherwise) and lowers to the same labelled-`for` constant-stack loop. The
`recur` intrinsic exists in `.gp` because a Go body is a statement sequence
where an implicit rewrite would be surprising; in an expression language
the call site *is* the loop-back edge. `total let rec` still runs the
structural-decrease proof first.

### 5.8 GADTs, existentials, typestate (v0.6 / v0.7)

```
type Expr : Type -> Type where
  | Lit  (v : Int)                          : Expr Int
  | BoolL (b : Bool)                        : Expr Bool
  | Add  (l : Expr Int)  (r : Expr Int)     : Expr Int
  | If   (c : Expr Bool) (t : Expr a) (e : Expr a) : Expr a
  | Wrap (inner : Expr a)                   : Expr (Slice a)

total let Eval (e : Expr a) : a :=
  match e with
  | Lit v      => v                          -- a ~ Int refined here
  | BoolL b    => b
  | Add l r    => Eval l + Eval r
  | If c t e   => if Eval c then Eval t else Eval e
  | Wrap inner => [Eval inner]

type Row (t : Type) :=
  | Packed {a : fmt.Stringer} (x : a) (tag : String)   -- bounded existential

type Socket : State -> Type where …                    -- typestate indices
let Send (s : Socket Open) : …
```

Same structural unification, same erasure walls (a composite result
matched at a bare parameter still can't name a Go case head — same guided
error), same bound-required rule for existentials.

### 5.9 Mutation, pointer-boxed ASTs, Go statements

```
@[box pointer]
type Node :=
  | Binary (left : Node) (right : Node)
  | Leaf   (value : Int)

let Simplify (n : Node) : Unit := do {
  match n with
  | Binary as b => do {                     -- binds the *Binary pointer
      b.left  := rewrite b.left;            -- rewrite: lowercase = private helper
      b.right := rewrite b.right
    }
  | Leaf _ => ()
}
```

`do` blocks are the honest Go-statement embedding: `let mut`, assignment,
`while`/`for … in`, `defer`, `go`, `return`. They lower to Go statement
sequences — this is the piece that keeps generated Go *idiomatic* rather
than closure-encoded, and it is the one genuinely new front-end surface
goml needs (the core lowering machinery already exists for `.gp`'s
expression-form hoisting, run in the opposite direction).

### 5.10 Methods for Go consumers (v0.1)

```
type Stack (a : Type) := { items : Slice a }

namespace Stack
  let Map (s : Stack a) (f : a -> b) : Stack b := …
end
```

`Stack.Map` lowers exactly like a `.gp` generic method: `StackMap` +
`//goplus:method` marker, so `.gp` callers write `s.Map(f)`, Go callers
`stack.StackMap(s, f)`, and goml callers `s.Map f` (dot notation resolves
through the head type's namespace, Lean-style).

### 5.11 Channels and select

```
let Run (inbox : Chan Msg) (done : Chan Unit) (acks : Chan Int)
        (s0 : State) : State := do {
  let mut s := s0;
  while true do {
    select with
    | m <- recv inbox => s := Step s m
    | _ <- recv done  => return s
    | _ <- send acks s.seq => ()          -- send arm
  }
}
```

`select` is a match-like expression over channel operations, lowered to a
native Go `select` statement (an optional `| default => …` arm makes it
non-blocking). Each `recv` arm binds the received value; `x, ok`-style
closed-channel observation composes as a tuple pattern.

## 6. Feature-parity map

| Go+ feature (version) | `.gp` spelling | goml spelling | Notes |
| --- | --- | --- | --- |
| Generic methods (v0.1) | `func (s Stack[T]) Map[U](…)` | `namespace Stack let map …` | same `//goplus:method` lowering |
| Enums/ADTs (v0.2) | `type … enum { A(x T) }` | `type … := A (x : T) \| …` | named fields kept for Go structs |
| Exhaustive match (v0.2) | `match`/`case` stmt | `match … with` expr | same Maranget checker |
| GADT pinning (v0.2/0.6) | variant result type | `where` ctor signatures | same unification/refinement |
| Multi-pattern arms (v0.14) | `case A(_), B(_):` | or-patterns, may bind uniformly | lowers to duplicated arms |
| Pipelines/composition (v0.3) | `\|>`, `>>>` | identical | precedence table shared |
| Partial application (v0.3) | `add(1, _)` | currying: `add 1` | placeholder dropped; lambda covers the rest |
| Railway (v0.4) | `?`, track-aware `\|>`, `>=>` | identical; plus `let*` (desugars to `Bind`) | `Result.of` at Go boundary |
| Expression control flow (v0.4) | expr `if`/`switch`/`match` + hoisting limits | everything is an expression | hoisting caveats mostly vanish (goml owns the whole lowering) |
| Typeclasses (v0.5) | `class`/`instance`/`law`, contextual | first-class keywords, `[C t]` binders, `@Witness` | near-1:1; already Lean-flavored |
| Folds/Cases (v0.6) | `Fold(e, Cases{…})` | `match` natively; derivation is output-only | Go consumers unchanged |
| Existentials (v0.6) | `Packed[A fmt.Stringer](…)` | `Packed {a : fmt.Stringer} (…)` | same bound-required, erase-at-bound |
| Delegation (v0.6) | `inner Store delegate` | `inner : Store @[delegate]` | same generated forwarders |
| QTT quantities (v0.7) | `0 n nat`, `1 f *os.File`, `m x ([]T)` | `{0 n : Nat}`, `(1 f : Ptr os.File)`, `(m x : Slice t)` | Idris2 notation; parenthesization wart gone |
| Totality (v0.7) | `total func` | `total let [rec]` | same structural-decrease proof |
| Indexed enums/typestate (v0.7) | `[n nat]`, tags, domains | `: Nat -> Type where`, `Socket Open`, `Region (Circle n) n` | same index-term language |
| Eq/refl (v0.7) | `0 p Eq[n, m]`, `refl` | `(0 p : n = m)`, `refl` | `=` as a type former |
| Linearity (v0.7) | `1 f T` ⇒ `Lin[T]` | `(1 f : t)` | identical erasure/cells |
| Tail calls (v0.21) | `tail func` + `recur(…)` | `@[tail] let rec`, plain calls | intrinsic dropped |
| Refinements (v0.22) | `type P refine(v int) { … }` | `type P := { v : Int \| … }` | same prover, guards, markers |
| Pointer boxing / repr | `//goplus:box pointer`, `//goplus:repr transparent` | `@[box pointer]`, `@[repr transparent]` | same registry entries |
| Derivation (v0.11) | default + `//goplus:derive` | `deriving Eq, Universe, Transform, Gen` (defaults match `.gp`) | same generated helpers |
| Laws → rapid tests (v0.5) | `//goplus:laws` knobs | `@[laws …]` attribute | same generation |
| Cross-package markers (v0.9) | `//goplus:` in `*_gp.go` | identical in `*_gml.go` | mixed packages legal |
| Go statements | native (superset) | `do` blocks (`defer`, `go`, `mut`, loops) + `select with` expression | the one new front-end surface |
| Struct tags | Go backtick literals | `@[json "port", yaml "port"]` field attributes | lowered to Go tag strings; checked syntax |
| LSP/editors (v0.9) | `goplus lsp` | same server, second grammar + sourcemap | completion still via gopls over generated Go |
| Java target (v0.141+) | `--target java` | unchanged | target selection is below the core |

## 7. How parity is achieved (staged plan)

**Architecture.** Milestone 0 does *not* build a second elaborator: the
goml parser desugars into the existing `.gp` surface AST (the cheapest
correct thing — currying flattens to multi-parameter functions when the
definition's arity is known, so `let f (a b : Int)` lowers to
`func F(a, b int)`, not a closure chain; partial application allocates a
closure exactly where `.gp`'s `add(1, _)` does today). Every checker,
decider, marker, and lowering is reused unchanged. A later milestone can
split a genuinely shared core AST out if the desugaring accumulates
strain, but parity never depends on that refactor.

**The parity gate is differential generation.** For each feature, the
executable spec gains a goml twin of the existing Godog scenario, and the
hard gate is: *the same program authored in `.gp` and `.goml` generates
byte-identical Go* (modulo the generated-file header and `_gp`/`_gml`
suffix). Determinism of `gen` is already a repo invariant, so this is a
`diff`, not a semantics argument. The end-to-end fixture: reauthor
`std/vec` and `std/option` in goml and diff their committed artifacts.

**Stages.**

- **M0 — the pure fragment.** Lexer/layout, `module`/`import`, `let`,
  simple sums, `match … with`, records (with `@[json …]` tag attributes),
  `|>`/`>>>`/`>=>`/`?`/`let*`, classes/instances/laws, `deriving`,
  attributes. Gate: goml twins of the v0.2–v0.6 feature files pass
  differentially.
- **M1 — the dependent surface.** Implicit binders and
  auto-quantification, quantities, `where`-form indexed types, `=`/`refl`,
  refinements, `total`, `@[tail]`. Gate: `std/vec` reauthored, identical
  artifact; the v0.7–v0.9 feature files twinned.
- **M2 — the Go embedding.** `do` blocks, `let mut`, `defer`/`go`,
  the `select with` expression, namespaces/dot notation, pointer-boxed
  matching, Go-interop typing rules (multi-result ⇒ tuples,
  trailing-`error` ⇒ `?`-eligible). Gate: one effectful std package
  (e.g. `std/clock` or `std/closeonce`) reauthored; the select lowering
  validated against a channel-using fixture.
- **M3 — tooling.** `goplus lsp` grows the second grammar (diagnostics
  from the real pipeline, sourcemap back through `*_gml.go`); `goplus fmt`
  for goml; and a bidirectional `goplus convert` (`.gp` ⇄ `.goml`) —
  mechanical in the M0 architecture because both sides share one AST —
  which doubles as a migration tool and a fuzzing oracle (round-trip =
  identity on the AST).

**Deliberate non-goals for v1 parity:** guards and literal patterns — they
land later as one shared-core milestone (exhaustiveness over literals needs
decider work) and surface in `.gp` (`case P if cond:`) and goml
(`| P if cond =>`) simultaneously; goml v1 parses them behind a flag and
rejects with a pointer to that milestone. Any semantics not already in the
core remains out of scope.

## 8. Idioms that make more sense in goml

1. **Erased indices become implicits.** `func First[T any](0 n nat, v
   Vec[T, n+1]) T` + `_ = t` ⇒ `let First : Vec a (n + 1) -> a | Cons h _
   => h`. The proof-only parameter vanishes from the page the same way it
   vanishes from the generated Go.
2. **Constructor instantiation is inferred.** `Err[U, E](e)` ⇒ `Err e`;
   `Nil[Pair[A, B]]()` ⇒ `Nil`. The `.gp` spellings exist because Go's
   call grammar demands the brackets somewhere; ML inference was built for
   exactly this.
3. **Quantities are binder annotations, not mystery integers.**
   `(0 n : Nat)`, `(1 f : t)` read as QTT; `0 n nat` reads as a syntax
   error you learn to love. The `m x ([]T)` parenthesization limitation
   disappears entirely.
4. **`recur` dissolves into tail position.** An expression language makes
   "the last thing evaluated" syntactically evident, so `@[tail]` is a
   *check*, not an intrinsic — and the defer/named-result caveats in the
   v0.8 delta don't arise because goml owns the whole body lowering.
5. **Contextual-keyword contortions end.** `enum`, `class`, `instance`,
   `law`, `delegate`, `total`, `tail`, `refine` become real keywords; the
   "match subjects may not start with `(`, `[`, `{`, or `<-`" caveat and
   the `type T class` puns are gone.
6. **Folds stop needing structs.** `<Enum>Cases` + `Fold` is the Go-shaped
   encoding of "functions per case"; in goml that is a `match` expression
   or a lambda per arm. The derivation survives purely as Go-consumer API.
7. **Laws read as algebra.** `law leftId (a : t) := combine empty a == a`
   — with `==` the derived structural equality — instead of
   `return reflect.DeepEqual(Combine(Empty(), a), a)`.
8. **Refinements are set comprehensions.** `{ v : Int | 0 < v && v <
   65536 }` is the textbook notation the feature implements.
9. **Currying subsumes placeholders.** `add 1` for `add(1, _)`; the
   placeholder grammar, its variadic restriction, and the `_.Method`
   deferral all become unnecessary.
10. **Or-patterns with uniform binders** strictly generalize v0.14's
    wildcard-only multi-pattern arms — the split-to-bind workaround goes
    away while the lowering (duplicated arms, per-alternative reachability
    rows) is unchanged.
11. **Directives become attributes.** `@[box pointer]`, `@[repr
    transparent]`, `@[laws Int, String]`, `deriving …` are checked,
    positioned syntax instead of magic comments — and they print in
    errors, hovers, and docs like the declarations they modify.
12. **Expression orientation removes the hoisting fence.** The v0.4 list
    of places `?`/expression-forms can't appear (for conditions, `&&`
    right-hand sides, case values, …) exists to avoid changing Go
    statement semantics around a hoist; goml lowers whole definitions, so
    the fence shrinks to the genuinely semantic cases (short-circuit
    evaluation order), which get defined behavior instead of a refusal.

## 9. Resolved decisions (2026-08-11)

Formerly the open-questions section; each was decided explicitly.

1. **Export rule: Go's case rule verbatim.** Capitalized = exported,
   lowercase = package-private, names travel into Go unchanged. No case
   mapping, no `private` keyword, no collision rule. ML-lowercase style is
   given up for public API in exchange for zero-surprise Go interop.
2. **Monadic bind: OCaml-style `let*`,** desugaring to method-aware `Bind`
   (`Result.Bind` today, a `Monad` class instance when the stdlib earns
   one). Core-expressible; no new semantics.
3. **`?` is kept alongside `let*`.** Each has a distinct home: `?` for
   terse Go-boundary calls (especially in `do` blocks, exact `.gp`
   parity), `let*` for structured pipeline code.
4. **Struct tags ship in v1 as field attributes** —
   `port : Int @[json "port", yaml "port"]` — lowered to Go backtick tag
   strings; checked syntax that prints in docs and hovers.
5. **`select` ships in v1 as a match-like expression** over channel
   operations (`| m <- recv ch => …`, `| _ <- send ch v => …`, optional
   `default`), lowered to a native Go `select`. Validated against a
   channel-using fixture in M2.
6. **Guards and literal patterns: core-first.** One shared-core milestone
   (decider work for literal exhaustiveness), then `.gp` and goml surface
   them simultaneously. goml v1 parses behind a flag and rejects with a
   pointer to that milestone.
7. **Extension: `.goml` / `*_gml.go`** (avoiding the `.gml` collision with
   GameMaker Language and Graph Modelling Language, which matters for the
   linguist/highlighting story the `.gp` repos already navigate).

## 10. Implementation status (v1, 2026-08-11 — goplus v0.144.0)

The front end follows the M0 architecture literally: `.goml` sources
transpile to `.gp` text and generate through the unchanged goplus
pipeline. There is no second elaborator.

**Shipped in v1:**

- `internal/goml` — lexer (`--` and nested `(* *)` comments),
  recursive-descent parser, and the `.gp` desugarer.
  `goforge.dev/goplus/goml` is the public facade (`Convert`, `Run`);
  `cmd/goml` is the binary (`gen [-check -stage]`, `convert [-o]`,
  `version`).
- Pipeline support: `gen.loadDir` discovers a `.goml` file when the
  facade supplies its transpiled text through the overlay (a bare
  `goplus gen` leaves `.goml` untouched); `emit.OutputPath` maps
  `foo.goml → foo_gml.go` and `foo_test.goml → foo_gml_test.go`; orphan
  scans map `_gml.go` back to `.goml`, and law tests from `.goml`
  sources survive plain goplus runs while their source exists. Because
  the output path equals the on-disk path, check/stage/stale-masking are
  inherited unchanged, and the generated header names the real source.
- The declarative surface: sum types (leading `|` required), GADT
  `where`-form with inferred index binder names, records with
  `@[delegate]` and `@[json "…"]` tag attributes, refinement
  comprehensions, aliases, **named propositions**
  (`type InRange (i : Nat) (n : Nat) := prop { And (Le 0 i) (Lt i n) }`
  lowering to the v0.156.0 `.gp` form; explicit binders only, no
  deriving — added 2026-08-30 under the gap program,
  `spec/gap-program-design.md`); classes (signature- and binder-form ops,
  defaults, laws, `extends`), named instances with **typed or bare
  members** (bare members infer their types from a locally declared
  class through its extends chain), `deriving`, `@[box pointer]` /
  `@[repr transparent]` / `@[derive …]` / `@[laws …]`, and
  **module-level attributes** (`@[laws "out=lawtest"] module algebra`).
- The dependent surface: explicit/implicit binders, `0`/`1` quantities
  and **multiplicity-variable quantities** (`{m : Mult} (m x : Slice t)`
  → `[m mult] … m x ([]t)`), auto-generalized type variables,
  nat-index inference (local-enum sorts, index arithmetic, `n = m` →
  `Eq[n, m]`), and index terms including **domain constructors and
  total calls** (`Region (Circle n) n` → `Region[Circle(n), n]`);
  `total let`; `@[tail] let rec` with tail-position self-calls lowered
  to `recur`.
- Bindings: a `let` with **no binders binds a package-level value**
  (`var Name [Type] = expr`), and `()` is the unit binder that keeps a
  nullary *function* spellable (`let main () := do { ... }`) — ML's rule,
  with no sniffing of result types or body shapes. Values cannot host
  hoisting forms or be generic; each case is a guided error.
- Data construction: **record literals** (`Settings { Port = p, Host = h }`,
  including the empty and package-qualified forms) lower to Go composite
  literals; `!` is logical negation. GADT headers accept type-sorted
  slots (`type Expr : Type -> Type where`) as well as nat-indexed ones,
  and a slot whose constructors pin every position concretely gets a
  synthesized name in its own sort (`Expr[a any]`).
- Expressions and clauses: `match with` (or-patterns, `as`-binding,
  unused binders print `_`), **multi-column clausal definitions**
  (comma rows, one constructor column), `if/then/else`, `let … ;`
  sequencing, **`let*` monadic bind** (desugars to a `?`-statement, so
  it works everywhere `.gp`'s `?` does), **match expressions in nested
  positions** (hoisted to a temporary before the enclosing statement),
  annotated lambdas, `|>`/`>>>`/`>=>`/postfix `?`, `@Witness`
  application.
- The Go embedding (M2): **`do` blocks** — `let` / `let mut`
  (multi-name, so Go multi-results destructure), assignment through
  locals and field paths (pointer-boxed mutation), `while … do`,
  `for … in … do` (range), `defer`, `go`, `return`, a final bare
  expression as the block result — and **`select with`** lowered to a
  native Go `select` (recv/send/default arms; arm bodies are
  statements). Hoisting guards reject match expressions where Go cannot
  host them (while conditions, select clauses) with pointed errors.
- Diagnostics: the transpiler records an emitted-line → source-line map
  (decl/statement grain); pipeline diagnostics attributed to a `.goml`
  file are remapped to their source line (`goml/goml_test.go`,
  `TestDiagnosticsMapToGomlLines`).
- The executable spec: `features/goml/goml.feature` runs the goml CLI
  through the Godog suite (generation, dependent erasure, do/select
  lowering, check mode, orphaning, coexistence with plain goplus runs,
  positioned errors). The differential gate — the same program authored
  both ways generates byte-identical Go modulo the header — runs in
  `goml/goml_test.go` for the pure and effectful fixtures.

**Deliberate deferrals (parse errors or absences today):** guards and
literal patterns (core-first, per §9 — both surfaces gain them from one
shared-core milestone), `open` (needs export knowledge), tuples and
list literals (not in the `.gp` core), unannotated lambdas (Go func
literals require types), Lean-style layout (two blunt rules instead:
application arguments start on the same line as the token before them,
and sums need a leading `|`), namespaces admit method lets only,
`goml fmt`, reverse conversion (`.gp → .goml`), and a dedicated goml
grammar for editors (the clients currently reuse the `.gp` one).
**Interface declarations shipped 2026-08-30** under the gap program
(`type Clock := interface { Now : Unit -> time.Time; io.Closer }` — a
curried member flattens to a Go method signature, a lone `Unit`
parameter to `()`, a `Unit` result is dropped, and a bare qualified
name embeds; multi-result methods still want a mixed package). `total`, `law`, and the other goml keywords are
reserved words (unlike `.gp`'s contextual claims) — `total` is not a
variable name.

## 11. The REPL (v0.145.0)

`goml repl` evaluates goml interactively. The decision that shapes
everything else: **there is no interpreter**. Writing one would be a
second implementation of the semantic core, which §1 exists to prevent.
Every input therefore transpiles the accumulated session, generates Go
through the ordinary pipeline, and runs it, so the REPL agrees with the
compiler by construction rather than by diligence.

**Decided consequences, each a real trade:**

- **Bindings re-execute on every evaluation.** Each evaluation is a
  fresh process, so a retained binding's initializer runs again. This is
  inherent to compile-and-replay and cannot be fixed without an
  interpreter. Mitigations: expression results are never retained (a
  bare effectful call runs exactly once), a binding whose body looks
  effectful is flagged when defined and marked `!` in `:list`, and the
  rule is stated in `:help`. Output capture was considered and declined:
  it would hide console writes while leaving the effects themselves —
  files, requests, clocks — untouched, which is worse than being honest.
- **Declarations never run the go tool.** `resolve.Backstop` is a full
  `go/types` check of the final texts, so a declaration that generates
  cleanly compiles. Measured, declarations land in ~100ms against
  ~450ms for expressions.
- **`:type` reports the declared signature** *(v0.147.0; it reported the
  erased Go type through v0.146.0)*. A named binding's signature is
  already written in goml spelling in the session's own source, so the
  REPL prints the parsed declaration back — `First : Vec a (n + 1) -> a`,
  indices and quantities intact — with no pipeline run at all. When the
  generated file is current, an `elaborated:` line adds the un-erased
  signature the pipeline recorded (`//goplus:dep`), which names the
  binders auto-quantification supplied. Anything with no declaration to
  read — an expression, an unannotated value, an imported name — falls
  back to type-checking the generated package, and that is now the only
  place the erasure caveat appears.
- **`it` expands inline.** A binding that referenced its own previous
  definition would be an initialization cycle, since the new definition
  replaces the old under the same name. So `it + 1` after `21 * 2`
  retains `let it := (21 * 2) + 1`. It is dropped when the expression is
  effectful, or when the accumulated text grows past 4KB.
- **Multi-line submits on a blank line.** Incompleteness is detected
  positionally (a parse error at end-of-input), but a clausal definition
  parses after its first clause, so a continuation needs an explicit
  end. `:{` … `:}` forces a block.
- **Instances are rendered with `@[laws off]`.** Law generation emits a
  test importing `pgregory.net/rapid`, which a session module does not
  require, and the directive is read per instance rather than per file.

## 12. Go interop (v0.146.0)

The `knockknock` example — two services authenticating machine-to-machine
over HTTP — was the forcing case for the operators real Go code needs.
Added: `&x` and `*p`; indexing `xs[i]`; channel send and receive outside
`select`; imported constructors in patterns (`result.Ok v`, where the
qualifier is lowercase and so cannot be told from a binder by case
alone); and `if` as a statement inside a loop body, where an empty
`else do { }` elides rather than emitting a dead block.

It also exposed two lowerings that produced wrong code instead of
refusing, both now fixed and regression-tested: a void function's
`if/else` fell through and ran both arms, and a match binder used only
inside a record literal or `do` block was blanked to `_`.

Deliberately still absent, because each wants syntax rather than a
guess: type conversions (`[]byte(s)`), `make`, and slice literals. Mixed
packages cover them — a `.go` file beside the `.goml` ones — which is the
documented escape hatch, and the example uses exactly three such
helpers. Reserved words now include `send`, `recv`, and `in`.

## 13. Typed holes (v0.147.0)

`?name` stands where code is not written yet, and generation reports what
belongs there: the goal type — in the un-erased dependent spelling where
the position is dependent — and the bindings in scope, including the
quantity-0 indices that the generated Go no longer mentions.

```
let Rest {0 n : Nat} (v : Vec a (n + 1)) : Vec a n :=
  ?rest

-- vec.goml:8:3: hole ?rest : Vec a n
--   erased: Vec a
--   in scope:
--     n : Nat (erased, quantity 0)
--     v : Vec a (n + 1)
```

The whole feature lives in the shared core — goml adds no inference. The
transpiler passes `?name` through to the `.gp` text verbatim, which is
also what makes the diagnostic exact: holes are *named*, so their source
positions are recorded by name and reported with a real column, where
ordinary diagnostics arrive at decl granularity with none.

The core necessarily computes the goal in the notation it works in
(`Vec[a, n+1]`, `[]string`), so goml **re-spells the answer** before
reporting it: `Vec a (n + 1)`, `Slice String`, `Map String Int`,
`Int -> String`, `Nat`. A dependent instantiation cannot be parsed as Go
— index lists take types and `n+1` is a term — so it is split textually
and its arguments spelled one at a time. Anything with no goml spelling
(a domain constructor such as `Region[Circle(n), n]`, a multi-result
function type) keeps the core's text: an approximate goal in the other
surface's notation still beats a mangled one.

Decided consequences:

- **A hole is spelled with a name, attached to the `?`.** That is what
  keeps it disjoint from postfix `?`: the try's `?` is glued to the
  expression before it, a hole's to the name after it. So `path ?;` is
  still a try and `f ?x` passes a hole. A spaced `? x` is a guided error
  rather than a guess.
- **Hole names are unique within a file**, which is what lets a goal be
  traced back to its exact `?` without a per-expression source map.
- **A declaration with a hole is not retained in the REPL.** Each
  evaluation compiles and replays the whole session, so a retained
  unfinished binding would fail every later input. Its goal is printed
  and `:holes` recalls it; retaining holed declarations would need the
  core to lower them to typed panic stubs, which is deliberately not part
  of this milestone.
- **Generation refuses to write while any hole remains** — including
  outside a module, where the goal itself cannot be computed. A committed
  `*_gml.go` therefore never contains a hole.
- **In the editor**, goals arrive as Information-severity diagnostics and
  hovering a `?name` serves the goal directly. Hover is answered natively
  rather than forwarded, because a hole is precisely the reason there is
  no generated Go to forward to; `goplus lsp` therefore advertises
  `hoverProvider` whether or not its gopls delegate started. `.goml`
  buffers are served too: the server runs the goml pipeline, so an
  unsaved buffer is transpiled in memory and its diagnostics come back
  positioned — and spelled — in goml. Delegated hover, definition, and
  completion remain `.gp`-only, because a `.goml` file's Go is generated
  from its transpiled `.gp` text and a direct source-to-output line map
  would be meaningless; that two-hop map is future work.
