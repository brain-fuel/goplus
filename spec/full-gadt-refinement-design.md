# Full GADTs and refinement types

This document records the implementation contract for the v0.9 language
milestone. It is normative alongside the executable feature suite.

## Gap in the current GADT implementation

The v0.6 structural unifier accepts arbitrary constructor result types, but
elimination is not full. Given:

```go
type Expr[T any] enum {
	Plain(value T)
	Wrap(inner Expr[T]) Expr[[]T]
}

func depth[U any](e Expr[U]) int {
	match e {
	case Plain(_): return 0
	case Wrap(inner): return 1 + depth(inner)
	}
}
```

the compiler currently rejects `Wrap`: `U` does not reveal the type argument
needed to spell the Go case head `Wrap[T]`. A wildcard only hides the problem;
it is not full GADT elimination.

## Full GADT requirements

1. Every structurally well-kinded constructor result may be matched through a
   generic scrutinee, including composite, repeated, permuted, and nested type
   arguments.
2. Matching introduces the constructor's equations and existential variables
   into the arm. Those facts type the entire arm, including nested matches,
   calls, returns, assignments, literals, and closures.
3. Exhaustiveness and reachability use the same unifier as arm typing. No
   constructor may disappear merely because Go cannot spell its instantiation.
4. Constructor fields retain their source types under the introduced equations;
   they do not degrade to user-visible `any`.
5. The generated package remains ordinary Go and exported APIs retain their
   authored generic signatures.
6. The behavior reconstructs from generated markers across package boundaries.

### Erased eliminator lowering

Go cannot instantiate a generic case head from a runtime type argument. The
portable lowering therefore needs an erased elimination path in addition to the
existing fast type-switch path:

- every variant exposes sealed, unexported discriminator and payload methods;
- a function containing an otherwise unspellable match receives an erased
  companion whose type-variable values are represented internally as `any`;
- the Go+ checker types that companion under the arm equations and inserts
  assertions only at proven boundaries;
- the authored generic function remains a typed facade and asserts the erased
  companion's result to its declared result type;
- renderable matches continue to use direct Go type switches.

This is not reflection-based field discovery: payload order and types are
generated from enum metadata, and the discriminator is sealed against foreign
implementations.

## Refinement requirements

The declaration syntax and proof/erasure rules are specified in
`grammar-v0.9.0.ebnf`. Refinements are propositions attached to values, not
nominal wrapper structs:

```go
type Port refine(value int) { 0 < value && value < 65536 }
```

- `Port(x)` is accepted only when the predicate follows from literals, current
  path facts, earlier refinements, and total boolean predicates.
- using a `Port` contributes its predicate to the local proof context;
- refined values erase transitively to the underlying Go type;
- exported parameter boundaries get runtime checks for ordinary Go callers;
- every Go+ return into a refined result is proved statically;
- GADT equations and refinement facts share one arm-local context, so either
  may discharge the other's obligations;
- generated markers carry the binder, base, and predicate for cross-package
  checking.

## Completion evidence

Completion requires passing executable scenarios for generic composite GADT
elimination, existential field use, nested and cross-package matches, positive
and negative refinement construction, control-flow narrowing, refined returns,
runtime boundary guards, composition with GADT arms, marker reconstruction, and
strict-Go parser equivalence. The full root and std test suites, race tests for
the affected compiler packages, `go vet`, generated-output checks, and
`git diff --check` must also pass.
