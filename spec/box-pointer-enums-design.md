# Pointer-boxed enums (`//goplus:box pointer`)

This document records the implementation contract for the pointer-boxed-enum
feature. It is normative alongside the executable feature suite
(`features/enums/box_pointer.feature`).

## Motivation

goplus enums lower to a sealed marker interface plus one struct per variant.
By default the marker method has a **value** receiver, so both `Variant` and
`*Variant` satisfy the interface, and `match` binds destructured field **values**
(copies). That is exactly right for small, immutable value enums.

It is the wrong shape for the most common hand-written Go sum type: a large node
that is **boxed behind a pointer** and **mutated in place** — abstract syntax
trees, intermediate representations, and the like. There, variants are stored as
`*Variant` (to avoid copying) and rewritten during passes (`e.Left = fold(e.Left)`).
Value-destructuring cannot express either fact: a pointer pattern does not parse,
and destructured field copies cannot be mutated. So the exhaustiveness that is
the whole point of `match` was unavailable on pointer-boxed sums — the ubiquitous
representation.

## The directive

```go
//goplus:box pointer
type Expr enum {
	Binary(Left Expr, Right Expr)
	Lit(Value int)
}
```

`//goplus:box pointer` on an enum declaration makes the enum **pointer-boxed**.
The default (no directive) is value boxing — the existing behavior, unchanged.

## Requirements

1. **Sealed by pointer.** Each variant's marker method is generated on a pointer
   receiver (`func (*Binary) isExpr() {}`), so only `*Binary` satisfies `Expr`.
   Construction is the ordinary Go `&Binary{…}`.
2. **Match binds the pointer.** A `match` over a pointer-boxed enum lowers its
   arms to `case *Variant:` type-switch heads; the case binder (`case b := Binary:`)
   is the `*Binary` scrutinee, so the arm may both read (`b.Left`) and mutate
   (`b.Left = …`) in place. A bare variant pattern needs no field binders (fields
   are reached through the bound pointer); `Variant(_, …)` field destructuring
   remains available and reads through the same auto-dereferencing pointer.
3. **Exhaustiveness unchanged.** Coverage runs on the enum's closed variant set
   exactly as for value enums: a `match` with no `case _:` must cover every
   variant, and a missing one is a compile error naming it
   (`non-exhaustive match on Expr: missing Lit`). A `case _:` opts out.
4. **No value-semantic derivation.** A pointer-boxed enum derives no `Fold`,
   `Cases`, `Equal`, `Traversal`, or erased-view helpers — those assume value
   semantics and are supplanted by match/type-switch consumption. `//goplus:box
   pointer` implies `//goplus:derive off` for these.
5. **Cross-package.** The `//goplus:box pointer` directive is preserved in the
   generated interface's marker group, so a consumer in another package that
   matches an imported pointer-boxed enum sees the pointer boxing and emits
   `case *pkg.Variant:` heads.
6. **Ordinary Go output.** The generated package remains ordinary Go: pointer
   receivers, `&Variant{}` construction, and a direct type switch. Nothing new is
   emitted at runtime.

## Non-goals

- No new surface syntax: `//goplus:box` is a directive and `case b := Variant:`
  already parses. The grammar is unchanged.
- Pointer boxing is per-enum, declared once; it is not a per-pattern or
  per-scrutinee choice, which would double the case space (a value could be
  boxed two ways). One canonical representation keeps exhaustiveness single
  case-per-variant.

## Completion evidence

Passing executable scenarios (`features/enums/box_pointer.feature`): pointer-only
interface satisfaction, mutation through the bound scrutinee, pointer-receiver
marker generation, non-exhaustive-match error naming the missing variant,
wildcard opt-out, and suppressed value-semantic derivation. The full root and
std suites, race on the affected compiler packages, `go vet`, generated-output
checks, and `git diff --check` also pass.

## Implementation

- `internal/directive/enum.go` — `ParseBoxMarker` for `//goplus:box`.
- `internal/gen/foldgen.go` — `enumBoxPointer(e)`; the fold/equal/traversal
  planners treat a boxed enum as derive-off.
- `internal/gen/enumgen.go` — thread `boxed` into `EnumSpec.BoxPointer` and
  `registry.Enum.BoxPointer`; skip the erased view for boxed enums.
- `internal/lower/enum.go` — `EnumSpec.BoxPointer` emits the marker on a pointer
  receiver.
- `internal/registry/enum.go` — read the preserved `//goplus:box pointer`
  directive into `Enum.BoxPointer` (cross-package).
- `internal/resolve/chain.go` — `rpatCaseType` prepends `*` to the case head for
  a pointer-boxed enum; the must-bind-fields rule is waived for boxed enums.
