# validate

`validate` is a Go+-authored typed validation algebra. Successful validation
returns `Validated[T,p]`, retaining the exact predicate as an erased index and a
sealed runtime witness for ordinary Go callers. Atomic rules compose through
ordered conjunction, and typed field projections preserve machine-readable
failure paths.

`IsValid` evaluates the rule's dedicated predicate path without materializing
`Failure` values. Both successful and unsuccessful boolean checks are
zero-allocation; callers that need diagnostics use `Check` or `Validate`.

`Pair[A,B]`, `PairOf`, and `Relate` provide a reflection-free cross-value
foundation for confirmation fields, bounds, ordering, and
`VarWithValue`-style rules. The relation remains an ordinary indexed `Rule`, so
successful pairs retain the exact predicate witness and compose with the same
validation algebra.

`Failure.Descriptor()` separates localization-neutral `Code`/`Param` semantics
from the field path and rendered error string. Applications can use the stable
descriptor as a translation key while keeping locale registries outside the
semantic core.

The semantic and compatibility contract is in
[`../../spec/validate-design.md`](../../spec/validate-design.md).

This package is independently structured. Compatibility tests and the external
adapter target `github.com/go-playground/validator/v10` v10.30.3, distributed
under the MIT license; see `LICENSE` for the preserved notice.
