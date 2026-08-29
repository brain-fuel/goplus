# Go+ (`goplus`)

Go+ is a language for authoring richer abstractions while emitting **portable,
idiomatic Go**. It is a **strict superset of Go**: every valid Go file is a
valid Go+ file (`.gp`), and every Go+ construct has a specific Go lowering.
Generated packages compile with the standard Go toolchain and may be
distributed and consumed **without** Go+ — the same interoperability story
Kotlin, Scala, and Clojure have with Java.

Go remains the default target. Go+ also has a deterministic **Java 25+**
backend for a checked portable subset: it emits Java source, a small versioned
runtime ABI, thin or bundled JARs, and optional strong JPMS modules. Target
selection is explicit and the same `.gp` package is elaborated separately with
`goplus_java`, `java25`, and `jvm64` build tags. See the
[Java 25 target guide](docs/java25.md) for commands, semantics, FFI contracts,
and the current compatibility ledger.

The package-rewrite program and opt-in dependent-typing sequence are tracked in
[GOALS.md](GOALS.md); its stable names are `/goals/01-decimal` through
`/goals/10-participle`.

## v0.155.0 — conjunction

`And[P, Q]` asserts both propositions, so the common precondition fits in
one parameter. Indexing wants `0 <= i` *and* `i < n`; saying that used to
take two proof parameters and two witnesses.

```go
func At[T any](0 i nat, 0 n nat, 0 p And[Le[0, i], Lt[i, n]], v Vec[T, n]) T {
	match v {
	case Cons(h, t):   // Lt[i, n] still prunes Nil: every part is a hypothesis
		_ = t
		return h
	}
}
```

It costs the decider nothing, for the same reason as everything else in
this sequence: `Decide` already takes a *list* of facts, and a conjunction
is exactly that. As a goal each part is proved in turn; in scope every
part becomes a hypothesis. It nests, and the parts may mix relations.

Erasure, the witnesses, the audit record, and match refinement all apply
unchanged — they were written against propositions rather than against
relations. A diagnostic substitutes the call's arguments into each part,
so you see which half failed: `cannot prove Le[0, 5] and Lt[5, 3]`.

## v0.154.0 — a bound refines a match

A proposition in scope became a hypothesis for the decider in v0.152.0,
so bounds compose at a call. It was still inert for **matching**:
`Lt[0, n]` did not tell an exhaustiveness check that `n` is non-zero, so
`Vec[T, n]` demanded a `Nil` arm that could never be taken.

```go
func Head[T any](0 n nat, 0 p Lt[0, n], v Vec[T, n]) T {
	match v {
	case Cons(h, t):   // Nil is impossible: the bound says n is positive
		_ = t
		return h
	}
}
```

For the third time in this sequence, the decider needed no new power.
Impossibility was decided by sign analysis over the index difference,
which cannot use facts; it now falls back to proving a strict inequality
in either direction, which is exactly what a hypothesis enables.

The generated boundary guard agrees, so a plain-Go caller passing the
impossible variant panics by name rather than computing garbage. And it
stays conservative: `Le[0, n]` does not prune (a natural may be zero), a
bound on an unrelated index proves nothing, and with no proposition in
scope nothing changes.

## v0.153.0 — a proof can no longer be skipped

**`Cast(v)` was accepted.** A call to a function with a proof parameter,
with the erased arguments left out, produced working Go with no proof, no
`assume`, and no audit record — so a dependent guarantee was bypassable by
accident and in silence. It reproduces against v0.146.0, so it long
predates the recent proposition work, and it undermined everything built
on it.

The cause: a proof argument is deleted by the same pass that discharges
it, so a call that never had one is textually identical to a call whose
argument has already been erased. The checker could not tell "never
proved" from "proved and erased", and assumed the second.

The ambiguity only exists *after* erasure begins. Obligations are now
settled on the first resolve iteration, where the text is still exactly
what you wrote — so the two cases are distinguishable with certainty
rather than by heuristic. The check deliberately uses no type
information, which is what lets it run that early: on the first iteration
a match arm's binders do not exist yet, so a type-directed check would
give up exactly where recursive dependent code lives.

```
main.gp:13:13: the proof argument for p of Cast cannot be omitted:
Eq[n, m] is a proposition, not an inferable index — pass refl (proved by
the decider) or assume (asserted on your authority)
```

**A proof-carrying function may also only be used in a direct call.**
Closing the omission alone would just relocate the bypass: `f := Cast`,
`Cast >>> g`, `v |> Cast`, and partial application all reach generated Go
with no proof. A proof can be written in exactly one place, so those are
refused. A dependent function with *no* proposition is unaffected and may
still be composed, piped, and stored.

Ordinary erased indices keep their inference — the check fires only for
parameters carrying a proposition, so `std` regenerates byte-identically.

One limit remains, inherent rather than open: a plain Go caller of the
generated artifact sees the erased signature and can call it with no
proof. Obligations are enforced at the `.gp` boundary.

## v0.152.0 — a bound in scope now means something

v0.151.0 made a bound statable but left it **inert**: a proposition in
scope did nothing for you. Two changes make it work.

**A proposition in scope is a hypothesis.** The enclosing function's own
proof parameters are given to the decider, so propositions compose —
under `Lt[1, n]`, both `Le[1, n]` and `Lt[1, n+1]` follow without a new
proof. The decider always took hypotheses; the proof path passed none.

**An erased index may be forwarded to a call.** A quantity-0 parameter
could not be named in any runtime position, and pass 1 cannot tell a call
argument from a runtime use — so an index could only ever be written as a
literal, which made composition unreachable:

```go
func Forward(0 n nat, 0 p Lt[1, n]) int {
	return NeedsLe(1, n, decide)   // n forwarded; 1 <= n from the bound
}
```

Passing an erased name to a genuine runtime parameter is still refused,
and that error now explains the erasure instead of only reporting an
undefined name.

### Proof arguments are now mandatory

Investigating v0.152.0 surfaced a bug that predated it: **omitting a proof
argument skipped its obligation.** `Cast(v)` was accepted with no proof
and no recorded assumption, so a dependent guarantee could be bypassed by
accident. v0.153.0 fixes it.

## v0.151.0 — bounds are statable

`Eq[a, b]` was the only proposition a signature could state. But the most
common dependent obligation is a **bound** — an index below a length —
and it could not be written down at all, so the programs that most want
dependent types could not express their real precondition.

`Le[a, b]` and `Lt[a, b]` complete the set:

```go
func At[T any](0 i nat, 0 n nat, 0 p Lt[i, n], v Vec[T, n]) T { … }

At(1, 3, decide, v)   // the decider discharges 1 < 3
At(5, 3, decide, v)   // cannot prove 5 < 3 at this call to At
```

The decider needed no new power — `FactGe` already sat beside `FactEq`.
Ordering was decidable long before it was sayable; this release makes it
sayable. Propositions erase exactly as `Eq` does.

**`refl` is reflexivity**, which is true of equality and meaningless for a
strict inequality, so it discharges `Eq` alone. `decide` is the general
witness — "the decider proved it" — and covers every proposition
including `Eq`, so one spelling now works across the set. `refl` remains
valid on `Eq`; using it on an ordering is an error that names `decide`.
`assume` asserts any of them, and the audit record carries the relation.

A proposition is not yet a *hypothesis*: `Lt[i, n]` in scope does not tell
a match that `n` is non-zero. Propositions as hypotheses, user-declared
predicates, and conjunction are the remaining Stage B work.

## v0.150.0 — assumptions cross the module boundary

`goplus assumptions` read the `.gp` source, so it could tell you what
*you* assumed but not what your dependencies assumed — and a consumer
receives committed Go, never the `.gp`. The audit stopped exactly at the
boundary where trust matters most.

An assumption now travels in the artifact, the way every other
cross-package fact in Go+ does:

```go
//goplus:dep Widen(v Vec[int, 2]) Vec[int, 3]
//goplus:assume Widen Cast p 2 = 3
func Widen(v Vec[int]) Vec[int] {
```

```
$ goplus assumptions ./...

in dependencies:
  example.com/lib: Widen assumes 2 = 3 for p of Cast

1 assumption(s): each is accepted on the author's authority, not proved.
```

The marker deliberately carries no source position: generated files are
committed and `gen -check`ed, and a position would churn the artifact
whenever an unrelated line moved.

## v0.149.0 — `assume`, and the decider stops being the last word

The arithmetic decider is sound but incomplete. Until now, a proposition
it could not discharge had no discharge at all: the error named both sides
and the workaround, and if neither applied, the program was unwritable.
There was no way to supply the fact yourself.

`assume` is that way. At a proof parameter, it stands where `refl` would:

```go
func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
	return v
}

Cast(1+1, 2, refl, v)    // the decider proved it
Cast(n*2, 2*n, assume, v) // you assert it
```

`refl` means *proved*; `assume` means *asserted, on your authority*. It
performs no check and erases exactly as `refl` does, so it never reaches
generated Go.

Because an assumption is the one place a dependent guarantee rests on a
claim rather than a proof, **every use is recorded**:

```
$ goplus assumptions ./...
main.gp:13:20: assumed 2 = 3 for p of Cast

1 assumption(s): each is accepted on the author's authority, not proved.
```

What a false assumption costs is bounded, and worth knowing before you
reach for one: it surfaces as a **panic at the erasure boundary, not a
wrong answer**. A quantity-0 index can never be used at runtime, so a
false index cannot flow into a computed value; every impossible-variant
conclusion it enables is backed by a generated guard — a named panic on
exported entry points, and the `default:` arm every lowered match keeps.
`assume` can cost you a crash at a known place. It cannot silently
corrupt a result.

It is deliberately not a diagnostic — that would block generation and
defeat the point. It is discoverable the way a Go reviewer finds `unsafe`:
by asking for it. The unprovable-equality error now names `assume` among
its remedies, so the hatch is found exactly where it is needed.

This opens GOALS.md Stage B (propositions and validated witnesses).
Propositions, predicate parameters, conjunction, and proof-preserving
functions remain ahead; so does a marker carrying assumptions into
distributed artifacts, so a consumer can audit what its dependencies
assumed.

## v0.148.0 — the editor speaks goml

`goplus lsp` now serves `.goml` buffers. The server runs the goml
pipeline, so an unsaved buffer is transpiled in memory and its
diagnostics — typed-hole goals included — come back positioned *and
spelled* in goml. Hovering a `?name` in a `.goml` file shows its goal.

Delegated hover, definition, and completion remain `.gp`-only: a `.goml`
file's Go is generated from its transpiled `.gp` text, so a direct
source-to-output line map would be meaningless. Hole goals need no map
and work in both surfaces.

This also fixes editor registration that never matched: the Neovim and
Zed clients keyed on a `.goplus` suffix — the language's name rather than
its file extension — so neither activated on a real `.gp` file. All three
clients (VS Code, Neovim, Zed) now register `.gp` and `.goml`.

## v0.147.1 — goml goals read in goml

The core necessarily computes a hole's goal in the notation it works in,
so a goml author was reading `.gp` spelling in their own surface. goml
now re-spells the answer before reporting it — `Vec a (n + 1)` for
`Vec[a, n+1]`, `Slice String` for `[]string`, `Int -> String` for
`func(int) string`, `Nat` for `nat`:

```
main.goml:8:3: hole ?rest : Vec a n
  erased: Vec a
  in scope:
    n : Nat (erased, quantity 0)
    v : Vec a (n + 1)
```

A dependent instantiation is split textually before parsing, because
`Vec[a, n+1]` is not valid Go: index lists take types, and `n+1` is a
term — precisely the Go+ extension. A shape with no goml spelling (a
domain constructor, a multi-result function type) keeps the core's text.

## v0.147.0 — typed holes

`?name` stands where code is not written yet, and generation answers with
the goal: the type that belongs there, un-erased where the position is
dependent, plus the bindings in scope — including the quantity-0 indices
the generated Go no longer mentions.

```go
func Rest[T any](0 n nat, v Vec[T, n+1]) Vec[T, n] {
	return ?rest
}

// main.gp:9:10: hole ?rest : Vec[T, n]
//   erased: Vec[T]
//   in scope:
//     n : nat (erased, quantity 0)
//     v : Vec[T, n+1]
```

Holes work in both surfaces and in every expression position, including a
package-level initializer. `?` in operand position claims a hole; the
v0.4.0 postfix `?` still claims the suffix, and the two never compete
because a hole's name is attached to its `?` (`a?b` is `(a?) b`, `a ?b`
applies a to a hole). **Generation refuses to write while any hole
remains**, so a committed `*_gp.go` never contains one.

In the editor, goals arrive as Information-severity diagnostics and
hovering a `?name` shows the goal — answered natively, since a hole is
exactly why there is no generated Go to forward a hover to.

In `goml repl`, a declaration with a hole prints its goal and is not
retained — each evaluation replays the whole session — and `:holes`
recalls the goals while you work.

**`:type` now reports the declared signature.** A named binding prints in
the goml spelling you wrote, indices intact, with no pipeline run:

```
goml> :type First
First : Vec a (n + 1) -> a
  elaborated: First[a any](0 n nat, v Vec[a, n+1]) a
```

Expressions, unannotated values, and imported names still report the
erased Go type, which is now the only place the erasure caveat appears.

## v0.146.0 — goml grows the Go-interop surface

Writing two networked applications in goml (the
[knockknock](https://github.com/brain-fuel/knockknock) example: two
services that authenticate machine-to-machine and then tell each other a
joke) forced out most of what real Go interop needs, plus two bugs that
generated wrong code rather than refusing.

**Two correctness fixes.** In a function with no result, an
`if … then … else …` dropped the `else` and ran *both* arms — the
then-arm does not return, so falling through was never sound. And a
match binder used only inside a record literal or a `do` block was
printed as `_`, so live code failed to compile with "undefined".

**New surface:** `&x` and `*p`; indexing `xs[i]`; channel send `ch <- v`
and receive `<- ch` outside `select`; `result.Ok v` and other imported
constructors in patterns; `if` as a statement inside loop bodies, where
an empty `else do { }` elides.

Still absent, and now documented: type conversions (`[]byte(s)`), `make`,
and slice literals. A mixed package — a `.go` file beside the `.goml`
ones — is the intended route until they land. `send`, `recv`, and `in`
are reserved words.

## v0.145.1 — the REPL prints correctly on a terminal

`goml repl` enters raw mode for history and cursor editing, which turns
off the terminal driver's newline translation. Its own writes did not
compensate, so every printed line left the cursor in the column it ended
on and output walked diagonally down the screen. Writes now carry the
carriage return themselves.

## v0.145.0 — the goml REPL, and a nullary `let` binds a value

`goml repl` evaluates goml interactively. There is no interpreter and
there will not be one: every input transpiles the accumulated session,
generates Go through the ordinary pipeline, and runs it, so the REPL
agrees with the compiler by construction.

```
goml> let Double (n : Int) : Int := n * 2
goml> Double 21
42
goml> let Port := 8080
goml> let Next : Int := Port + 1
goml> Next
8081
goml> :type Double
func(n int) int
```

Declarations accumulate; `:list`, `:undo`, `:drop`, `:load`, and `:save`
manage them, `:type` reports the (erased) Go type, and `:gp`/`:go` show
the lowering and the generated Go. Multi-line input continues while it is
incomplete and submits on a blank line; imports of common standard
packages are added on first use; `it` is the last result.

The honest caveat, stated in `:help`: because each evaluation compiles
and runs the whole session, **retained bindings re-execute every time**.
Expression results are never retained, so a bare effectful call runs
exactly once, and a binding that looks effectful is flagged when you
define it. Declarations skip the run entirely — the pipeline's type
check is the same one the compiler performs — which is why they land in
about a tenth of the time an expression takes.



goml follows ML's rule: **binders present means a function, none means a
value**. `let Port : Int := 8080` now lowers to `var Port int = 8080`, so
bindings compose (`let Next : Int := Port + 1`); previously every `let`
became a function, making `Port + 1` a type error and leaving
`let Port := 8080` generating invalid Go. A nullary *function* takes the
unit binder, exactly as OCaml spells it:

```
let Answer := 42                 -- var Answer = 42
let main () := do { ... }        -- func main() { ... }
let Boot () : Unit := start ()   -- func Boot() { start() }
```

A package-level value cannot host statements or be generic, so a `match`,
expression-`if`, `?`, `let*`, `select`, or free type variable in a value
body is a guided error naming both fixes (`()` or a parameter list) rather
than a silent fallback. Values initialize in Go's dependency order.

## v0.144.1 — goml surface completion

Writing a full worked tutorial against v0.144.0 surfaced three holes,
each now closed: a type-indexed GADT header (`type Expr : Type -> Type
where`) parses (previously only nat-indexed kinds did), records can be
**constructed** as well as declared (`Settings { Port = p, Host = h }`
lowers to a Go composite literal), and logical negation `!` exists.
Synthesized index-parameter names now follow their slot's sort, so a
fully-pinned GADT reads `Expr[a any]` rather than `Expr[n any]`.

## v0.144.0 — goml: an ML-family surface

Go+ gains a second front end. goml (`.goml`) is an SML/OCaml/Idris2/
Lean4-flavored surface for the same semantic core: sources transpile to
`.gp` and generate through the unchanged pipeline, emitting committed,
ordinary Go (`<file>_gml.go` beside the source, same `//goplus:`
markers). One core, two surfaces, one output — packages may mix `.gp`
and `.goml`, and consumers cannot tell which surface authored them.

```
type Vec (a : Type) : Nat -> Type where
  | Nil : Vec a 0
  | Cons (head : a) (tail : Vec a n) : Vec a (n + 1)

let First : Vec a (n + 1) -> a       -- {0 n : Nat} auto-quantified
  | Cons h _ => h                    -- Nil impossible at n+1

let Load (path : String) : Result Config Error := do {
  let raw := os.ReadFile path ?;     -- postfix ? at the Go boundary
  parse raw
}
```

The v1 surface covers sums, GADT `where`-form, records (tag
attributes), refinement comprehensions, classes/instances/laws
(instance members may omit types when the class is local), the
dependent core (implicit binders, QTT quantities including multiplicity
variables, `n = m` propositional equality, constructor and total-call
index terms), `total` and `@[tail]` lowering, `let*` monadic bind,
multi-column clausal definitions, hoisted match expressions, `do`
blocks (`let mut`, field assignment, `while`/`for … in`, `defer`,
`go`), and `select with` lowered to native Go select, record literals, and `!`.
Pipeline diagnostics map back to `.goml` source lines. `goml gen` drives
generation (`-check`/`-stage` as in goplus); `goml convert` prints the
`.gp` lowering. The design, decisions, and parity map live in
[spec/goml-design.md](spec/goml-design.md); the executable spec twins
live under `features/goml/`.

## v0.143.0 — Java artifact production

`go tool goplus build --target java ./...` produces the primary JAR, sources
JAR, standard-doclet Javadoc JAR, and a deterministic
`goplus.java.build/v2` manifest. Build configuration is schema v2 and contains
only compiler and unsigned artifact paths; Maven coordinates, credentials,
signing, POM generation, and network publication belong to assayxport.

Publish with `ax publish` after the matching Go version is publicly available.
`goplus publish` is retained only as an actionable compatibility error.

## v0.141.1 — Deterministic Java 25+ Target

Go remains the default backend, and `goplus gen`, `build`, `test`, and `run`
now accept `--target java` for the checked portable subset documented in
[the Java 25 target guide](docs/java25.md). The backend emits deterministic
Java source and versioned runtime artifacts, compiles with `javac --release
25`, creates reproducible thin or bundled JARs, and optionally emits strong
JPMS modules. Java declarations are indexed from the selected JDK,
classpath/module path, and multi-release JARs; unsupported Go semantics fail
with source-mapped diagnostics rather than silently changing behavior.

The public `compiler` package exposes independently versioned artifact sets,
while `goplus.toml` records explicit targets and Java artifact policy. CI runs
the Java integration suite on JDK 25. The stdlib version line is independent
and remains `std/v0.210.0`. The v0.141.1 patch also separates the distribution
version reported by the CLI and LSP from the generated-source compatibility
vintage, so installed binaries report the module release accurately without
forcing an unrelated stdlib regeneration.

## v0.28.0 — Solver-Driven Representation and Existential Foundations

Go+ now supports opt-in `//goplus:repr transparent` lowering for monomorphic
single-variant enums. Indexed wrappers retain dependent markers and exhaustive
match semantics while generated Go uses a concrete alias, eliminating
interface boxing. True sum types and existential variants cannot select this
representation.

Unbounded existential variant parameters may now use `any`; authored fields
retain their shared hidden-type relationship for constructor checking while
generated Go erases compositions containing that existential. Variadic indexed
arguments participate fully in inferred-index consistency and result recovery.
Natural indices nested below ordinary generic arguments are also preserved and
checked across package boundaries; a value such as
`Term[UninterpretedSort[3]]` cannot satisfy a parameter requiring
`Term[UninterpretedSort[1]]` after outer-type erasure.
Cross-package GADT matching now evaluates fixed result types in the enum's
package even when the consumer spells them through an import qualifier. This
lets compatibility packages exhaustively recognize standard-library term
variants without weakening GADT reachability checks.
Together these features support `std/smt`'s sorted terms, context-indexed
immutable solvers, models, proofs, assumptions, unsat cores, adaptive Boolean
solving, exact integer difference logic, and ground congruence closure.
Built-in-sort-indexed unary and binary uninterpreted functions now include
`Int -> Int` and `Int × Int -> Int`; wrong-sort applications are
unrepresentable in Go+, while `std/smtlib` decides ground `QF_UFLIA`
congruence over symbols and exact integer constants.

## v0.27.0 — Grammar-Indexed Parser Foundations

Go+ now validates omitted natural witnesses against every indexed runtime
argument, including imported calls. A shared `nat` can no longer appear as one
identity in the first argument and another in the second merely because its
proof parameter was inferred and erased. The Participle rewrite forced this
cross-package consistency check while preserving concise call sites and
portable generated Go.

## v0.26.0 — Method-Indexed Linear HTTP Foundations

The `/goals/09-resty` forcing case adds cross-package inference for omitted
quantity-0 natural indices when the same call also consumes a quantity-1
capability. Method, body replayability, and decode phase can now travel through
an erased request type without making Go+ callers spell proof-only arguments or
ordinary-Go `LinOf` wrappers. Stable local index recovery is ordered by source,
so chained indexed transitions retain concrete results.

`std/retry` now exposes the protocol-neutral pieces shared by its original
consumer and the HTTP client rewrite: normalized attempt counts, cancellable
waiting with a zero-allocation immediate path, and overflow-safe capped delay
progression. HTTP status policy and response-body ownership remain outside the
standard library.

## v0.25.0 — Dependent Rewrite Foundations

`std/decimal` provides immutable arbitrary-precision base-10 arithmetic,
precision-safe JSON and database boundaries, six exhaustive rounding modes,
and division results that distinguish exact, rounded, and zero-divisor cases.
Its `Precision` and `Scale` refinements guard generated Go entry points.

The dependent surface retains scale in `Fixed[p]`: addition and subtraction
require equal indices, multiplication computes `Fixed[p+q]`, and rescaling is
an explicit lossy transition. Cross-package checking now reconstructs indices
through nested calls, single-assignment locals, and dependent parameters;
reassigned indexed locals require an explicit rebind or rescale. Generated Go
retains a sealed runtime scale witness, so erased callers receive corresponding
boundary protection.

## Collection Algebra and Dependent Shapes

`std/nonempty` provides owned sequences with total head, last, and reduction;
it is shared by `std/algebra` and the `goforge.dev/lo` compatibility module.
`std/iter` is the lazy sequence algebra: a `Seq[T]` wraps the standard
`iter.Seq[T]` with fluent combinators (`Map`, `Filter`, `FilterMap`, `FlatMap`,
`Uniq`, `Take`/`Drop`/`TakeWhile`/`DropWhile`, `Reverse`, `Chunk`, `Concat`,
`Fold`) and `Seq()` erases back to the standard iterator at any boundary.
Fallible pipelines carry `std/result.Result` elements (`TryFold`,
`CollectResults`); its laws (round-trip, `map∘filter` fusion, `take`/`drop`
partition, reverse involution) are property-checked. The `goforge.dev/lo/it`
iterator package is a thin facade over `std/iter`, so the engine and its laws
have a single home.
`std/vec` now adds equal-length `Zip`, `Fin[n]` bounds evidence, and total
`At`. Constructor-produced indices survive long enough for cross-package
checking: different vector lengths and out-of-range evidence are rejected
before erasure, while generated Go retains runtime shape guards.

## Typed Configuration Snapshots

`std/config` resolves defaults, remote values, files, environment, flags, and
overrides into immutable `Snapshot[s]` values that retain provenance.
`Key[T,s]` prevents keys from one schema being read from another; `Require`
produces presence evidence, and `Subset[s,sub]` is the checked route to a
reindexed projection. The `goforge.dev/viper` migration facade consumes this
representation while preserving a bounded Viper v1.21.0 API tier.

## Pattern-Indexed HTTP Routes

`std/http/route` ties parsed `Pattern[p]` values to `Request[p]`, typed
parameter keys, sealed handlers, route-set fingerprints, and explicit
middleware capabilities. Cross-package checking rejects a key or handler from
another pattern before erasure. The `goforge.dev/chi` facade compiles familiar
Chi registration into immutable route snapshots with structured ambiguity
diagnostics and OpenAPI-ready flat metadata.

## Typed Expressions and Verified Bytecode

The sibling `goforge.dev/expr` module is the `/goals/06-expr` forcing case for
GADTs, finite existential parse results, equality transport, and indexed
bytecode. Its Go+-authored `Expr[T]` core has explicit effects and evaluation
failures; `Instruction[input,output]` and `Stack[n]` make underflowing programs
unrepresentable. Cross-package fixtures prove valid `0 -> 1 -> 2 -> 1`
composition and reject underflow or a false `Eq[n,m]` witness before erasure.
The ordinary-Go facade pins Expr v1.17.8 and publishes an explicit language and
617-declaration API matrix rather than implying compatibility with deferred
dynamic features.

## Schema-Aware JSON Paths

The sibling `goforge.dev/gjson` module is the `/goals/07-gjson` forcing case for
`Path[S,T]`, presence-indexed `Lookup[T,p]`, finite existential paths, and
schema-preserving composition. Validated immutable documents return borrowed
zero-copy views with a documented lifetime; byte input is owned, numbers retain
lossless decimal spelling, JSON-lines traversal streams, and modifiers live in
explicit immutable registries. Cross-package fixtures reject schema mismatch
and use of missing evidence as present. One typed integer path is consumed by
both JSON and `std/cbor`, without prematurely promoting a shared std package.

## v0.24.1 — Durable Workflows and Effect Boundaries

Six Go+-authored standard-library packages make command-line and delivery
workflows explicit without pretending external systems are transactional:

- `std/process` executes commands with capture/stream modes, cancellation,
  typed exit failures, and secret-safe diagnostics.
- `std/semver` implements strict Semantic Versioning 2.0 parsing, precedence,
  formatting, and major/minor/patch increments.
- `std/workflow` journals resumable saga steps and compensations; its supplied
  file journal uses crash-safe replacement.
- `std/config` composes defaults, format adapters, semantic validators, and
  field-path errors.
- `std/fsatomic` replaces files through write, sync, rename, and directory
  sync, with platform-correct durability behavior.
- `std/cas` defines typed observations and the exhaustive `Updated`, `Changed`,
  and `Missing` outcomes shared by compare-and-swap stores.

These packages deliberately separate locally provable workflow state from
facts that must be observed again at an external mutation boundary. Go+
enums make CAS outcomes exhaustive, while generated Go keeps every package
usable by ordinary Go programs.

The v0.24.1 patch expands compact control flow for consistent analysis across
Go 1.26 hosts and gives workflow journal records an explicit stable JSON schema.

## v0.23.0 — QUIC v2, CBOR, and Proven DAG-CBOR

The zero-configuration HTTP client now discovers HTTP/3 through Alt-Svc and
safely falls back through HTTP/2 and HTTP/1.1. Package-level `Get`, `Head`,
`Post`, and `PostForm` need no client setup. The default server supports RFC
9368 compatible QUIC version negotiation and RFC 9369 QUIC v2.

The standard library adds a generic `serde.Codec[T]` surface, deterministic
RFC 8949 CBOR, explicit RFC 7049 canonical compatibility, streaming sequences,
tags, raw messages, custom marshal hooks, and diagnostic notation.

Strict DAG-CBOR enforces the IPLD data model, deterministic encoding, string
map keys, finite 64-bit floats, and CID tag 42. `Proof[T]` witnesses that input
is the unique canonical DAG-CBOR representation of the requested Go or Go+
type, retaining immutable bytes and their SHA-256 digest.

## v0.22.0 — Refinement Types and Structural GADT Elimination

Refinement declarations add checked semantic subsets of existing Go types:

```go
type Positive refine(value int) { value > 0 }
```

Go+ proves constant constructions, path-guarded values, calls, assignments,
and returns where possible, rejects unproved uses at generation time, and
emits runtime guards at exported Go boundaries. Refinement contracts survive
package boundaries through generated markers while the emitted representation
remains ordinary Go.

Structural GADT matches can now eliminate composite indices through generic
scrutinees, including recursive cases such as `Expr[U]` matching a constructor
whose result is `Expr[[]T]`. Generated private erased views preserve the typed
public facade and keep the output portable to the standard Go toolchain.

## v0.21.0 — Native Tail Calls

`recur(nextArgs...)` is an explicit self-tail-call statement. It evaluates
the next parameter values left-to-right, rebinds them simultaneously, and
starts the function body again without growing the Go stack:

```go
tail func sumTo(n, acc uint64) uint64 {
	if n == 0 {
		return acc
	}
	recur(n-1, acc+n)
}
```

The generated Go is a labelled `for` loop with parameter assignment and
`continue`; no recursive call remains. `recur` is valid only as the final
statement of a function or tail branch, its arity is the function parameter
count, variadic state is passed as its slice without `...`, and method
receivers remain fixed. Arguments use Go assignment evaluation order. Because
the form explicitly requests loop semantics, one invocation has one defer
stack (each executed `defer` still registers on it) and named results persist
between iterations. Inside a `total func`, the same structural-decrease proof
applies before lowering.

## v0.14.0 — Multi-Pattern Arms

Driven by rune's elaborate/store rewrite — rigidity and spine
classifiers union constructors in one arm:

```go
match t {
case Var(_), Ref(_), Univ(_), Prop:  // one arm, four constructors
	return true
case App(_, _):
	return false
}
```

Alternatives take only wildcard arguments and the arm cannot bind the
value (split the arm to bind); every alternative is its own
reachability row, so a redundant alternative is an unreachable-arm
error and alternatives count toward exhaustiveness.

## v0.11.0 — Deep Structure

The release that arms the rune kernel rewrite: every enum's recursive
structure is now derivable, not hand-rolled.

```go
// Self-recursive enums derive deep traversals (descent sees through
// binder wrappers like Scope{Name string; Body Tm} and slices):
for sub := range TmUniverse(t) { … }        // t + all subterms, preorder
t2 := TmTransform(t, simplify)              // bottom-up rewrite, copies slices

// Monomorphic enums derive structural equality with per-variant hooks —
// proof irrelevance is an override on a derived base, not a hand-written
// walk (handled=false falls through to the derived comparison):
irrelevant := TmEqOverrides{Cast: func(x, y Cast) (bool, bool) {
	return TmEqual(x.A, y.A) && TmEqual(x.B, y.B) && TmEqual(x.X, y.X), true
}}
TmEqualWith(a, b, irrelevant)

// std/option joins std/result: Of/Get at the comma-ok boundary,
// IsSome/IsNone, Map, Bind, UnwrapOr, OrElse.
```

Traversals and equality are nil-tolerant (optional fields like an
elective type annotation pass through untouched), func/map/chan content
makes equality silently underivable (closures have no structure), and
variant doc comments now survive lowering onto the generated structs —
generated Go documents itself on pkg.go.dev.

## v0.10.0 — The Dogfood Rewrite

[goforge.dev/cadence](https://goforge.dev/cadence) v0.2.0 is authored
in Go+ — the first external artifact rewritten in the language — and
the rewrite drove three features home:

```go
// Derived rapid generators for every enum (emission is demand-driven —
// law tests, or //goplus:derive gen — so rapid never enters go.mod uninvited):
s := GenStrategy(rt)

// Laws quantify over enums, drawn through the derived generator:
type Interpreter[T any] class {
	Serve(host T, r Region, s Strategy, ctx RequestContext) (Tree, error)
	law Fallback(host T, s Strategy) { … }
}
// → every instance gets a generated rapid property; violations shrink
//   to counterexamples like m={-1}

// Operations declare multiple results; tests are Go+ too:
//   foo_test.gp → foo_gp_test.go (still a _test.go to the go tool)
```

In cadence, Strategy became a real sum (illegal states unrepresentable),
the hand-rolled FallbackHolds died, and the fallback law is now part of
the Interpreter class — proven automatically for every interpreter
anyone writes.

## v0.9.0 — Tooling: LSP, Editors, go generate

No language changes — this release is about living with goplus:

- **`goplus lsp`** ships inside the binary (one version, always in
  lockstep): diagnostics as you type from the real gen pipeline run in
  memory, plus hover, goto-definition, and completion delegated to
  gopls over the generated Go and mapped back through the sourcemap.
  The server's dispatch layer is authored in goplus itself.
- **Editors**: VS Code (marketplace-ready extension), Neovim, Zed, and
  GoLand/IntelliJ (platform LSP API) — all thin clients of `goplus lsp`;
  see editor/.
- **go generate is canonical**: `goplus init` scaffolds
  `//go:generate go tool goplus gen ./...`; the workflow is
  `go generate ./... && go build ./...`, with the goplus wrapper as
  convenience.
- **Cross-package hardening**: generated files carry a `//goplus:v`
  vintage stamp (a newer file tells you the exact upgrade command);
  marker reconstruction is package-wide; index domains cross packages
  (`Socket[s states.State]`); imported Eq propositions unfold their
  callee's totals; and a missing instance names the transitive package
  that provides one.

## v0.8.0 — Parser Combinators (std/parsec)

```go
import "goforge.dev/goplus/std/parsec"

// A complete arithmetic evaluator: precedence, parens, whitespace.
func grammar() parsec.Parser[int] {
	var expr parsec.Parser[int]
	factor := parsec.Label(parsec.Or(number(), parsec.Between(parsec.Symbol("("), parsec.Defer(&expr), parsec.Symbol(")"))), "expression")
	term := parsec.Chainl1(factor, mulOp())
	expr = parsec.Chainl1(term, addOp())
	return parsec.Then(parsec.Spaces(), parsec.Before(expr, parsec.EOF()))
}

v, err := parsec.RunString(grammar(), "(1+2)*3")   // 9
// errors carry positions and labels:
// 1:3: unexpected '*', expecting expression
```

Parsec-style consumed/empty semantics: `Or` commits once a branch has
consumed input, `Try` restores the lookahead — the discipline that
keeps performance predictable and errors precise. Input STREAMS from
any io.Reader: the buffer retains only what a live `Try` could rewind
to, split UTF-8 runes decode across read boundaries, and a
byte-at-a-time reader parses identically to a string (rapid-tested,
along with the monad identities, Or associativity, and
Try-never-consumes). The library is goplus eating its own cooking: Reply
is a goplus enum matched in every combinator, its derived Fold consumes
replies without a match, and Run's output rides the v0.4 railway.
Also in this release: the linear-value cell is atomic
(`CompareAndSwap`), so even racing double-users get exactly one winner.

## v0.7.0 — The Dependent Core

```go
// Length-indexed vectors: the index is real, checked, and erased.
type Vec[T any, n nat] enum {
	Nil() Vec[T, 0]
	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
}

// 0-quantity parameters exist only at check time:
func Head[T any](0 n nat, v Vec[T, n+1]) T {
	match v {
	case Cons(h, t): // Nil is impossible at n+1 — no other arm needed
		return h
	}
}

// Compiler-verified termination; callable in types:
total func Plus(a, b nat) nat {
	if a == 0 { return b }
	return Plus(a-1, b) + 1
}

// Propositional equality, discharged by the decider:
func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
	return v
}
w := Cast(1+1, 2, refl, v) // proves 1+1 = 2; erases to the identity

// Linearity — consumed exactly once, statically AND at runtime:
func Process(1 f *os.File) error { return f.Close() }
```

goplus now carries a real dependent core: quantities (QTT's 0/1/ω plus
multiplicity variables), total functions with structural termination
and guarded nat subtraction, enums indexed by nats, enum tags
(typestate: `Socket[Open]`), and structured first-order data
(`Region[Circle(n), n]`), a normalization-by-evaluation engine where
`n+m ≡ m+n` is definitional, and a sound linear-arithmetic decider
that prunes impossible match arms, discharges `refl` proofs, and
justifies subtraction. Everything erases: indices vanish from the
generated Go, exported dependent functions grow precise runtime guards
for plain-Go callers, and linear values travel as generated use-once
Lin[T] cells that panic on reuse. `std/vec` ships the length-indexed
sequence. Where the decider cannot prove an obligation, the error names
both sides and the workaround — stuck-with-guidance, never silent.

## v0.6.0 — Folds, Full GADTs, Existentials, Delegation

```go
// Structural GADT result types — any type expression per position:
type Expr[T any] enum {
	Wrap(inner Expr[T]) Expr[[]T]
	Flipped(a A, b B) Duo[B, A]      // cross-position
}

// Bounded existentials, erased at the boundary:
type Row[T any] enum {
	Packed[A fmt.Stringer](x A, tag string)
}

// Every enum derives a one-level fold (opt out: //goplus:derive off):
n := Fold(Some(7), OptionCases[int, string]{
	Some: strconv.Itoa,
	None: func() string { return "-" },
})

// Kotlin-style interface delegation:
type Logged struct {
	inner Store delegate   // Logged implements Store; override by declaring
	log   *log.Logger
}
```

GADT result arguments are now arbitrary type expressions, resolved by
structural unification: possibility filtering, case heads, constructor
inference, and refinement all work through composites and cross-position
uses, and refinement wraps EVERY mismatched conversion boundary in an
arm (naked returns included) — only actual mismatches wrap. Where Go's
erasure cannot name a case head (a composite argument matched at a bare
type parameter), the arm is a guided error and `case _:` covers it.
Existential type variables must carry an interface bound — Go cannot
express a match arm generic in a hidden type — and erase to that bound
in fields, constructors, and binders. `std/result` now ships a derived
`result.Fold(r, result.ResultCases[T, E, R]{Ok: …, Err: …})`.

## v0.5.0 — Typeclasses

Lean-flavored classes, named instances, implicit dispatch, and a
law-tested algebraic hierarchy — all lowering to plain Go witness
structs a Go consumer can call directly:

```go
type Monoid[T any] class {
	Semigroup[T]
	Empty() T
	law LeftId(a T) { return reflect.DeepEqual(Combine(Empty(), a), a) }
}

instance IntAdd Group[int] {
	Combine(a, b int) int { return a + b }
	Empty() int { return 0 }
	Invert(a int) int { return -a }
}

func Accumulate[T Monoid](xs []T) T {
	acc := Empty()
	for _, x := range xs {
		acc = Combine(acc, x)
	}
	return acc
}

Accumulate([]int{1, 2, 3})   // one Monoid[int] instance in scope: found implicitly
```

A class is an algebraic structure in the mathematical sense: a carrier
set (the instantiating type) together with operations on it, satisfying
declared laws — a `Monoid` is the triple (T, `Combine`, `Empty`) with
associativity and a two-sided identity, and an instance names one
concrete such structure. That is why int has TWO monoids, and why
implicit resolution refuses to pick between them: with `std/algebra`
imported, `Accumulate([]int{…})` is ambiguous between `IntAdd` (a Group,
by subsumption) and `IntMul`, and the error names both. You disambiguate
by naming the structure you mean:

```go
algebra.Accumulate(algebra.IntAdd, []int{2, 3, 4})  // 9  — the additive monoid
algebra.Accumulate(algebra.IntMul, []int{2, 3, 4})  // 24 — the multiplicative monoid
```

Explicit witnesses subsume exactly like implicit dispatch (v0.6.1):
`IntAdd` is a `Group` instance, `Accumulate` wants a `Monoid`, and the
compiler inserts the upcast — you name the structure, never the
coercion.

A class lowers to a flat witness struct (`Monoid[T]` with `func` fields);
an instance to a package value; a class constraint to a leading witness
parameter that call sites receive implicitly. Classes embed to form
hierarchies (diamonds collapse; upcasts are generated), operations may
carry **default bodies** instances can omit, and a **stronger instance
satisfies a weaker constraint** (a `Group[int]` instance answers a
`[T Monoid]` call). Ambiguity is a hard error naming the candidates; the
escape hatch is calling the lowered signature directly. `law` members
declare boolean properties over the operations, and **law tests generate
by default** for every concrete instance (rapid properties, inherited
laws included) with `//goplus:laws` knobs (`off`, `[int] [string]`
instantiations for generic instances, `gen=`, package-level `out=`).
`goforge.dev/goplus/std/algebra` ships the Magma→Group hierarchy, stock
instances, and `Accumulate`/`FoldMap`.

## v0.4.0 — Typed Failure

Railway-Oriented error handling in the Wlaschin style: a shipped
`Result[T, E]` library, track-aware pipelines, Kleisli composition,
postfix `?` propagation, and expression-oriented control flow.

```go
import "goforge.dev/goplus/std/result"

// Track-aware |>: once a Result flows, stages lift by shape —
// T→Result binds, T→(U, error) adapts, T→U maps, T→() tees (Ok only),
// dot segments see the raw Result. Err bypasses everything.
n := s |> validate |> strings.TrimSpace |> strconv.Atoi |> audit |> .UnwrapOr(0)

// Kleisli >=>: compose fallible steps; plain steps lift, the rail
// opens at the first step that can fail, Err short-circuits.
pipeline := strings.TrimSpace >=> validate >=> strconv.Atoi >=> save
// (value, error) functions adapt automatically: strconv.Atoi >=> double

// Postfix ?: propagate failure to the enclosing function.
data := os.ReadFile(path)?          // (…, error) enclosing: zero-value early return
n := parse(s)?                      // Result enclosing: returns Err, typed errors preserved

// Expression-oriented if / switch / match — arms are single expressions.
y := if x > 2 { "big" } else { "small" }
grade := switch score {
case 10:
	"A"
default:
	"B"
}
area := match shape {
case Circle(r):
	3.14 * r * r
case Rect(w, h):
	w * h
}
```

`goforge.dev/goplus/std` is a nested module with **zero dependencies**,
written in Go+ and shipped as generated Go (`go get goforge.dev/goplus/std`).
`Result[T any, E error]` carries typed failures; `Of` enters the railway
from a Go-shaped `(value, error)` pair, `Unpack` leaves it. `?` works with
Result values, `(…, error)` calls, and bare errors, in both Go-shaped and
Result-shaped functions. Expression forms hoist to statements before their
anchor — hoisted sites evaluate before the rest of their statement, in
source order — and a match expression gets the full v0.2.0 machinery:
exhaustiveness, GADT refinement, nested patterns.

## v0.3.0 — Functional Flow

Pipelines, composition, partial application, and placeholders — all
lowering to the plain Go you would have written:

```go
total := xs |> Filter(isEven) |> Map(double) |> Sum
// Sum(StackMap(StackFilter(xs, isEven), double))

answer := 21 |> Some |> .Map(double).UnwrapOr(0)

toStr  := double >>> strconv.Itoa      // func(int) string
inc    := add(1, _)                    // partial application
between:= clamp(_, lo, hi)             // placeholder anywhere in a call
```

`x |> f(a)` inserts the piped value as the first argument (a placeholder
`_` picks a different slot); bare-name segments are **method-aware**: they
resolve against the piped value's members — full Go selector semantics
plus Go+ generic and enum methods — and against functions, constructors,
builtins, and conversions in scope. Resolving to *both* is a hard error
naming the two explicit spellings (`.Map(f)` for the member, `Map(_, f)`
for the function). Multi-result stages follow Go's spread rule
(`"42" |> strconv.Atoi |> handle` when `handle(int, error)`). `>>>`
composes left-to-right into a capture-once closure, constructor operands
included (`double >>> Some`). Partials capture their callee and fixed
arguments exactly once at creation, method receivers bind-time.

## v0.2.0 — Algebraic Data Types

Sum types with exhaustive pattern matching, constructor generation, and
initial GADT support — lowered to sealed interfaces plus variant structs
that plain Go consumes with an ordinary type switch:

```go
// option.gp
package option

type Option[T any] enum {
	Some(value T)
	None
}

func (o Option[T]) Map[U any](f func(T) U) Option[U] {
	match o {
	case Some(v):
		return Some(f(v))
	case None:
		return None
	}
}
```

`match` is exhaustive: a missing variant is a compile error with a witness
(`non-exhaustive match on Shape: missing Rect(_, _)`), checked by Maranget
usefulness so nested patterns like `Add(Lit(a), Lit(b))` are covered
correctly. Constructors infer their type arguments from arguments or the
expected type (`var o Option[int] = None`), auto-wrap into closures in
function position (`xs.Map(Some)`), and qualify (`Option[int].None`) when a
name is genuinely ambiguous. GADT variants may pin their result type —
`Lit(v int) Expr[int]` — excluding impossible arms and refining type
parameters inside matching arms (the classic typed interpreter works).
Emitted enums carry `//goplus:enum`/`//goplus:variant` markers, so importing
packages get constructors, matching, and exhaustiveness from the committed
Go artifact alone.

```go
// emitted
type Option[T any] interface{ isOption(T) }

type Some[T any] struct{ Value T }
func (Some[T]) isOption(T) {}
// … plain-Go consumer:
switch v := o.(type) {
case option.Some[int]:
	fmt.Println(v.Value)
}
```

## v0.1.0 — Generic Methods

Methods may introduce type parameters not present on their receivers:

```go
// stack.gp
package stack

type Stack[T any] struct{ items []T }

func (s Stack[T]) Map[U any](f func(T) U) Stack[U] {
    out := Stack[U]{items: make([]U, 0, len(s.items))}
    for _, x := range s.items {
        out.items = append(out.items, f(x))
    }
    return out
}
```

`goplus gen` emits `stack_gp.go` beside the source — committed to your repo,
protobuf-style — lowering each generic method to a package-level generic
function:

```go
// Code generated by goplus from stack.gp. DO NOT EDIT.

//goplus:method (Stack[T]) Map[U]
func StackMap[T any, U any](s Stack[T], f func(T) U) Stack[U] { … }
```

Go+ callers keep method syntax — `s.Map(strconv.Itoa)` — including chained
calls, explicit instantiation (`s.Map[string](f)`), method values
(`f := s.Map[string]`), and promotion through embedded fields. Plain-Go
consumers of your published package call `stack.StackMap(s, strconv.Itoa)`.
The `//goplus:method` marker makes the emitted file self-describing, so packages
that import yours get method syntax too — even when your package is fetched
as ordinary Go with `go get`.

## CLI

```
# Canonical workflow: the go toolchain drives, goplus only generates.
goplus init                 # scaffold //go:generate wiring (flag: -hook)
go get -tool goforge.dev/goplus/cmd/goplus@latest   # pin goplus in go.mod (Go 1.24+)
go generate ./...        # regenerate *_gp.go from *.gp
go build ./...           # plain Go from here (test/vet/run likewise)

# Convenience wrapper: same thing, one word shorter.
goplus gen ./...            # generate *_gp.go from *.gp
goplus gen -check ./...     # exit 1 if any generated file is stale (CI)
goplus gen -stage ./...     # regenerate and git-add results (pre-commit)
goplus assumptions ./...    # list propositions accepted with assume
goplus build|test|run|vet   # generate, then delegate to the go tool
goplus version

# The ML-family surface: .goml sources emit <file>_gml.go via the same pipeline.
goml gen ./...              # transpile *.goml and generate *_gml.go
goml gen -check ./...       # exit 1 if any generated file is stale (CI)
goml repl                   # evaluate goml interactively
goml convert file.goml      # print the .gp lowering
goml version
```

## Install

```
go install goforge.dev/goplus/cmd/goplus@latest
go install goforge.dev/goplus/cmd/goml@latest    # the ML-family surface
```

The standard library is a separate, dependency-free module:

```
go get goforge.dev/goplus/std@latest
```

## Keeping generated code fresh

Use the [pre-commit](https://pre-commit.com) framework:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/brain-fuel/goplus
    rev: v0.1.0
    hooks:
      - id: goplus-gen
```

When outputs are stale, the first `git commit` attempt regenerates **and
stages** the fixed files, then aborts (pre-commit's behavior for any hook that
modifies files); retry the commit and it passes. `goplus-check` is a
verify-only variant for CI.

## Specification

The spec is executable: the Godog/Cucumber feature suite under
[`features/`](features/) plus the grammar deltas in
[`spec/`](spec/) (one EBNF per milestone). Run it with `go test ./...`.

## Limitations (by design)

- A lowered generic method is a function, not a method — it cannot help a
  type satisfy an interface. This is fundamental to the lowering (Go
  interfaces cannot express generic methods) and will not change.
- Uninstantiated generic method values (`f := s.Map`) are errors, matching
  Go's rule for uninstantiated generic function values.
- Match subjects may not start with `(`, `[`, `{`, or `<-` (those spellings
  stay valid Go); bind such subjects to a variable first. Literal patterns
  and guards arrive in a later milestone.
- v0.2.0 GADT result-type arguments are the enum's own type parameter or a
  ground named type per position; refinement applies to `T`-typed returns
  (use `any(x).(T)` manually elsewhere).
- `|>` and `>>>` are the lowest-precedence operators; `xs |> len > 0`
  parses as `xs |> (len > 0)` and gets a parenthesize hint. Placeholders
  cannot stand for variadic parameters, and `_.Method` receivers wait for
  a later milestone.
- `?` and expression if/switch/match lower by hoisting statements, so they
  cannot appear where an early return or eager evaluation would change
  semantics: for conditions/post statements, else-if conditions, the right
  side of `&&`/`||`, case values, select communications, assignment
  left-hand sides, whole deferred/go calls, or package level (each is a
  guided error).

## Roadmap

| Version | Theme |
| ------- | ----- |
| v0.1.0  | Generic methods — shipped |
| v0.2.0  | Algebraic data types, exhaustive matching — shipped |
| v0.3.0  | Pipelines, composition, partial application — shipped |
| v0.4.0  | Typed failure: std/result, railway pipes, Kleisli `>=>`, postfix `?`, expression-oriented control flow — shipped |
| v0.5.0  | Typeclasses: classes, instances, implicit dispatch, laws, std/algebra — shipped |
| v0.6.0  | Folds, structural GADTs, bounded existentials, delegation — shipped |
| v0.7.0  | The dependent core: QTT quantities, total functions, indexed enums, Eq, linearity, std/vec — shipped |
| v0.8.0  | std/parsec: streaming parser combinators — shipped |
| v0.9.0  | Tooling: goplus lsp + four editors, go generate canonical, cross-package hardening — shipped |
| v0.10.0 | The dogfood rewrite: cadence v0.2.0 in Go+; derived generators, laws over enums, multi-result ops, Go+ tests — shipped |
| v0.11.0 | Deep structure: derived traversals (Children/Universe/Transform), derived structural equality with overrides, std/option, variant doc preservation — shipped |
| v0.13.0 | The standard library grows nine: kleene, latch, clock, guarded, deepmap, retry, registry, memo, closeonce (from the envoy-go rewrite) — shipped |
| v0.14.0 | Multi-pattern match arms — shipped |
| v0.15.0 | Generalized typed failures and release hardening — shipped |
| v0.16.0 | Cross-package class laws retain qualified type imports in generated tests — shipped |
| v0.16.1 | Cross-package law imports support both grouped and single import declarations — shipped |
| v0.17.0 | RFC 6455 and RFC 7692 WebSockets with Go+ protocol states, exhaustive conformance, and optimized framing — shipped |
| v0.17.1 | WebSocket completion audit: linear capabilities, full handwritten coverage, broader performance gates, and protocol hardening — shipped |
| v0.18.0 | RFC 8441 WebSockets over HTTP/2 with transparent RFC 6455 fallback and stream multiplexing — shipped |
| v0.20.0 | Native RFC 9000 QUIC, RFC 9114 HTTP/3, RFC 9220 WebSockets, and H3 → H2 → H1.1 fallback — shipped |
| v0.21.0 | Explicit `tail func` / `recur` lowering to constant-stack Go loops — implemented |
| v0.22.0 | Refinement types and structural GADT elimination — shipped |
| v0.23.0 | QUIC v2, CBOR, serde, and proven DAG-CBOR — shipped |
| v0.24.0 | Process, SemVer, durable workflows, validated config, atomic files, and CAS — implemented |
| v0.24.1 | Cross-host analyzer compatibility and stable workflow-journal JSON — implemented |
| v0.25.0 | Goals 01–08 dependent rewrite foundations: indexed decimal, collections, config, HTTP routes, expressions, JSON paths, validation, and schedules — shipped |
| v0.27.0 | Goal 10 foundations: consistent inferred indices across all imported runtime arguments — shipped |
| v0.26.0 | Goal 09 foundations: inferred preserved indices across linear calls and shared overflow-safe retry primitives — shipped |
| v0.155.0 | `And[P, Q]` conjunction: the whole precondition in one proof parameter — shipped |
| v0.154.0 | A proposition in scope refines a match, pruning variants its bound excludes — shipped |
| v0.153.0 | Proof arguments are mandatory, closing a bypass that predated v0.146.0; a proof-carrying function may only be used in a direct call — shipped |
| v0.152.0 | Propositions in scope act as hypotheses; erased indices may be forwarded to calls — shipped |
| v0.151.0 | `Le`/`Lt` propositions and the general `decide` witness: bounds are statable — shipped |
| v0.150.0 | Assumptions travel in the generated artifact via `//goplus:assume`, so a consumer can audit what its dependencies assumed — shipped |
| v0.149.0 | `assume`: an auditable escape hatch for propositions the decider cannot discharge, with `goplus assumptions` — shipped |
| v0.148.0 | `goplus lsp` serves `.goml` buffers; editor clients register both surfaces — shipped |
| v0.147.1 | goml re-spells hole goals into goml notation — shipped |
| v0.147.0 | Typed holes: `?name` goals with un-erased dependent types and in-scope bindings; goml `:holes` and declared-signature `:type` — shipped |

## License

MIT — see [LICENSE](LICENSE).
