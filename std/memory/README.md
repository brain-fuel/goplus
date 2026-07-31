# Go+ region memory

`memory` is the backend-neutral runtime foundation for explicit bulk lifetime
management. Native builds reserve storage outside the traced Go heap; WASM and
Plan 9 use reusable linear/managed storage.

The first release deliberately exposes generation-checked byte handles rather
than pointers. A retained byte slice cannot outlive an arena because reads and
writes copy through the handle. `Delete`, `Rollback`, and `Reset` invalidate
handles immediately. Secure zeroing is the default.

Future Go+ compiler releases will layer lexical lifetime indices, linear owned
values, borrowed references, stack-placement proofs, and
`//goplus:layout soa` over this representation.

## Allocation groups

`Arena.Group` creates an explicit ownership boundary over related allocations.
`Group.Reset` securely releases every member while keeping the group reusable;
`Group.Release` permanently closes the capability. Handles released by either
operation remain invalid even when their spans are reused.

## Structure-of-arrays layouts

`SoA2` and `SoA3` provide typed columnar storage and deterministic conversion
from and back to `Row2` and `Row3` AoS values. `Reset` clears logical contents
while retaining capacity for hot-path reuse; `Release` clears and drops the
backing columns.
