Feature: Propositional equality
  Eq[a, b] is a proposition: a proof parameter (always quantity 0)
  discharged at call sites by `refl` through the arithmetic decider,
  with the callee's other index parameters bound to the caller's
  arguments. Everything erases — Eq, refl, and the parameters carrying
  them leave no trace in the generated Go. An equality the decider
  cannot prove is an error naming both sides after substitution.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: refl discharges ground and symbolic equalities
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func Swap(0 n nat, 0 m nat, 0 p Eq[n+m, m+n]) string {
      	return "commutes"
      }

      func main() {
      	v := Cons(1, Cons(2, Nil[int]()))
      	w := Cast(1+1, 2, refl, v)
      	_ = w
      	fmt.Println("cast ok")
      	fmt.Println(Swap(3, 4, refl))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains:
      """
      cast ok
      commutes
      """
    And the file "main_gp.go" contains:
      """
      func Cast[T any](v Vec[T]) Vec[T] {
      """
    And the file "main_gp.go" contains:
      """
      	w := Cast(v)
      """
    And the file "main_gp.go" contains:
      """
      	fmt.Println(Swap())
      """

  Scenario: An unprovable equality is an error naming both sides
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func main() {
      	v := Cons(1, Nil[int]())
      	_ = Cast(1, 2, refl, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove 1 = 2 at this call to Cast; the arithmetic decider could not discharge refl"

  Scenario: A proof argument must be refl or assume
    Given a Go+ file "main.gp":
      """
      package main

      func Claim(0 n nat, 0 p Eq[n, n]) string {
      	return "ok"
      }

      func main() {
      	x := 1
      	_ = Claim(1, x)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "the proof argument for p of Claim must be refl (proved by the decider) or assume (asserted on your authority)"

  Scenario: A proof parameter must be erased
    Given a Go+ file "main.gp":
      """
      package main

      func Bad(0 n nat, p Eq[n, n]) string {
      	return "no"
      }

      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "a proof parameter (Eq[n, n]) must be erased: give p quantity 0"

  Scenario: assume asserts a proposition the decider cannot discharge
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func Widen(v Vec[int, 2]) Vec[int, 3] {
      	return Cast(2, 3, assume, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    # The proof argument erases; only the audit marker mentions it.
    And the file "main_gp.go" does not contain "Cast(2, 3, assume"
    And the file "main_gp.go" contains "//goplus:assume Widen Cast p 2 = 3"

  Scenario: assumptions are listed for review, positioned in the source
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func Widen(v Vec[int, 2]) Vec[int, 3] {
      	return Cast(2, 3, assume, v)
      }
      """
    When I run goplus with arguments "assumptions ."
    Then the exit code is 0
    And stdout contains "main.gp:13:20: assumed 2 = 3 for p of Cast"
    And stdout contains "1 assumption(s)"

  Scenario: a package with no assumptions says so
    Given a Go+ file "main.gp":
      """
      package main

      func Double(n int) int { return n * 2 }
      """
    When I run goplus with arguments "assumptions ."
    Then the exit code is 0
    And stdout contains "no assumptions"

  Scenario: the unprovable-equality error names assume as the escape hatch
    Given a Go+ file "main.gp":
      """
      package main

      func Claim(0 n nat, 0 m nat, 0 p Eq[n, m]) string {
      	return "ok"
      }

      func main() {
      	_ = Claim(1, 2, refl)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "or assert it with assume"

  Scenario: Each assumption is reported at its own line
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func A(v Vec[int, 2]) Vec[int, 3] {
      	return Cast(2, 3, assume, v)
      }

      func B(v Vec[int, 5]) Vec[int, 7] {
      	return Cast(5, 7, assume, v)
      }
      """
    When I run goplus with arguments "assumptions ."
    Then the exit code is 0
    And stdout contains "main.gp:13:20: assumed 2 = 3 for p of Cast"
    And stdout contains "main.gp:17:20: assumed 5 = 7 for p of Cast"
    And stdout contains "2 assumption(s)"

  Scenario: A false assumption panics at the boundary rather than corrupting a result
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """
    And a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Head[T any](0 n nat, v Vec[T, n+1]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func main() {
      	empty := Nil[int]()
      	lying := Cast(0, 1, assume, empty)
      	fmt.Println(Head(0, lying))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 1
    And stderr contains "goplus: Head: v with index n+1 cannot be Nil"

  Scenario: An assumption travels into the generated artifact for consumers
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      // Widen keeps its own doc comment.
      func Widen(v Vec[int, 2]) Vec[int, 3] {
      	return Cast(2, 3, assume, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    And the file "main_gp.go" contains "// Widen keeps its own doc comment."
    And the file "main_gp.go" contains "//goplus:assume Widen Cast p 2 = 3"
    When I run goplus with arguments "gen -check ."
    Then the exit code is 0

  Scenario: A bound is statable, and the decider discharges it
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Bounded[T any](0 i nat, 0 n nat, 0 p Lt[i, n], v Vec[T, n]) int {
      	return 0
      }

      func AtMost[T any](0 i nat, 0 n nat, 0 p Le[i, n], v Vec[T, n]) int {
      	return 1
      }

      func Ok(v Vec[int, 3]) int {
      	return Bounded(1, 3, decide, v) + AtMost(3, 3, decide, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    # Propositions erase exactly as Eq does.
    And the file "main_gp.go" contains "func Bounded[T any](v Vec[T]) int"

  Scenario: A false bound is refused, naming the relation
    Given a Go+ file "main.gp":
      """
      package main

      func Bounded(0 i nat, 0 n nat, 0 p Lt[i, n]) int {
      	return 0
      }

      func main() {
      	_ = Bounded(5, 3, decide)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove 5 < 3 at this call to Bounded"
    And stderr contains "or assert it with assume"

  Scenario: refl is reflexivity, so an ordering wants decide
    Given a Go+ file "main.gp":
      """
      package main

      func Bounded(0 i nat, 0 n nat, 0 p Lt[i, n]) int {
      	return 0
      }

      func main() {
      	_ = Bounded(1, 3, refl)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "the proof argument for p of Bounded must be decide (proved by the decider) or assume (asserted on your authority)"

  Scenario: decide discharges an equality too, so one witness covers every proposition
    Given a Go+ file "main.gp":
      """
      package main

      func Claim(0 n nat, 0 m nat, 0 p Eq[n, m]) int {
      	return 0
      }

      func main() {
      	_ = Claim(1+1, 2, decide)
      	_ = Claim(1+1, 2, refl)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0

  Scenario: An assumed bound is audited with its own relation
    Given a Go+ file "main.gp":
      """
      package main

      func Bounded(0 i nat, 0 n nat, 0 p Lt[i, n]) int {
      	return 0
      }

      func Use() int {
      	return Bounded(5, 3, assume)
      }
      """
    When I run goplus with arguments "assumptions ."
    Then the exit code is 0
    And stdout contains "assumed 5 < 3 for p of Bounded"

  Scenario: An erased index may be forwarded to another dependent call
    Given a Go+ file "main.gp":
      """
      package main

      func NeedsLe(0 i nat, 0 n nat, 0 p Le[i, n]) int { return 0 }

      func Forward(0 n nat, 0 p Lt[1, n]) int {
      	return NeedsLe(1, n, decide)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    And the file "main_gp.go" contains "func Forward() int"

  Scenario: A proposition in scope is a hypothesis, so bounds compose
    Given a Go+ file "main.gp":
      """
      package main

      func NeedsLe(0 i nat, 0 n nat, 0 p Le[i, n]) int { return 0 }

      func NoBound(0 n nat) int {
      	return NeedsLe(1, n, decide)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    # Without Lt[1, n] in scope there is nothing to derive 1 <= n from.
    And stderr contains "cannot prove 1 <= n at this call to NeedsLe"

  Scenario: An erased parameter used as a value still says why
    Given a Go+ file "main.gp":
      """
      package main

      func Plain(x int) int { return x }

      func Misuse(0 n nat, 0 p Le[0, n]) int {
      	return Plain(n)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "a quantity-0 parameter exists only at check time"

  Scenario: A proof argument cannot be omitted
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func Sneak(v Vec[int, 2]) Vec[int, 3] {
      	return Cast(v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gp:13:13: the proof argument for p of Cast cannot be omitted"
    And stderr contains "Eq[n, m] is a proposition, not an inferable index"
    And stderr contains "assume (asserted on your authority)"
    And the file "main_gp.go" does not exist

  Scenario: A proposition with no runtime arguments cannot be omitted either
    Given a Go+ file "main.gp":
      """
      package main

      func Swap(0 n nat, 0 m nat, 0 p Eq[n+m, m+n]) string {
      	return "ok"
      }

      func Use() string {
      	return Swap()
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "the proof argument for p of Swap cannot be omitted"
    And the file "main_gp.go" does not exist

  Scenario: An omitted proof is caught inside a match arm
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func Inside(v Vec[int, 2]) int {
      	match v {
      	case Cons(h, t):
      		_ = t
      		_ = Cast(v)
      		return h
      	case Nil():
      		return 0
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "the proof argument for p of Cast cannot be omitted"
    And the file "main_gp.go" does not exist

  Scenario: An ordering obligation names decide as its witness
    Given a Go+ file "main.gp":
      """
      package main

      func Bounded(0 i nat, 0 n nat, 0 p Lt[i, n]) int {
      	return 0
      }

      func Use() int {
      	return Bounded()
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "pass decide (proved by the decider) or assume"

  Scenario: An omitted erased index is still inferred
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Length[T any](0 n nat, v Vec[T, n]) int {
      	match v {
      	case Cons(h, t):
      		_ = h
      		return Length(t) + 1
      	case Nil():
      		return 0
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    And the file "main_gp.go" contains "func Length[T any](v Vec[T]) int"

  Scenario: A proof-carrying function cannot be taken as a value
    Given a Go+ file "main.gp":
      """
      package main

      func Claim(0 n nat, 0 m nat, 0 p Eq[n, m]) string {
      	return "ok"
      }

      func Take() {
      	f := Claim
      	_ = f
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "Claim carries a proof obligation (p Eq[n, m]) and can only be used in a direct call"
    And the file "main_gp.go" does not exist

  Scenario: A proof-carrying function cannot be a pipeline stage
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m] {
      	return v
      }

      func P(v Vec[int, 2]) Vec[int, 2] {
      	return v |> Cast(1+1, 2, refl)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot be a pipeline stage"
    And the file "main_gp.go" does not exist

  Scenario: A proof-carrying function cannot be partially applied
    Given a Go+ file "main.gp":
      """
      package main

      func Claim(0 n nat, 0 m nat, 0 p Eq[n, m], tag string) string {
      	return tag
      }

      func Partial() {
      	_ = Claim(1+1, 2, refl, _)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot be partially applied"
    And the file "main_gp.go" does not exist

  Scenario: A dependent function with no proposition may still be a value
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Length[T any](0 n nat, v Vec[T, n]) int {
      	return 0
      }

      func Use() int {
      	f := Length[int]
      	return f(Nil[int]())
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go

  Scenario: A bound in scope rules out an impossible variant
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Head[T any](0 n nat, 0 p Lt[0, n], v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    # The boundary guard agrees with the check: a plain-Go caller passing
    # Nil panics by name rather than computing garbage.
    And the file "main_gp.go" contains:
      """
      panic("goplus: Head: v with index n cannot be Nil")
      """

  Scenario: Without a bound the same match is non-exhaustive
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Head[T any](0 n nat, v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "non-exhaustive match on Vec[T]: missing Nil"

  Scenario: A bound too weak to exclude zero does not prune
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Head[T any](0 n nat, 0 p Le[0, n], v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "non-exhaustive match on Vec[T]: missing Nil"

  Scenario: A bound on an unrelated index proves nothing
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Head[T any](0 n nat, 0 m nat, 0 p Lt[0, m], v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "non-exhaustive match on Vec[T]: missing Nil"

  Scenario: A conjunction states the whole precondition in one parameter
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func At[T any](0 i nat, 0 n nat, 0 p And[Le[0, i], Lt[i, n]], v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }

      func Ok(v Vec[int, 3]) int {
      	return At(1, 3, decide, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    # Every part is a hypothesis, so Lt[i, n] prunes the Nil arm.
    And the file "main_gp.go" contains:
      """
      panic("goplus: At: v with index n cannot be Nil")
      """

  Scenario: A conjunction fails when either part does, and says which
    Given a Go+ file "main.gp":
      """
      package main

      func At(0 i nat, 0 n nat, 0 p And[Le[0, i], Lt[i, n]]) int {
      	return 0
      }

      func Bad() int {
      	return At(5, 3, decide)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove Le[0, 5] and Lt[5, 3] at this call to At"
    And the file "main_gp.go" does not exist

  Scenario: An assumed conjunction is audited whole
    Given a Go+ file "main.gp":
      """
      package main

      func At(0 i nat, 0 n nat, 0 p And[Le[0, i], Lt[i, n]]) int {
      	return 0
      }

      func Asserted() int {
      	return At(5, 3, assume)
      }
      """
    When I run goplus with arguments "assumptions ."
    Then the exit code is 0
    And stdout contains "assumed Le[0, 5] and Lt[5, 3] for p of At"

  Scenario: A named proposition abbreviates a precondition
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      type InRange[i nat, n nat] prop { And[Le[0, i], Lt[i, n]] }

      func At[T any](0 i nat, 0 n nat, 0 p InRange[i, n], v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	}
      }

      func Ok(v Vec[int, 3]) int {
      	return At(1, 3, decide, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    # The declaration names facts, never values, so it erases completely
    # and only its marker survives for consumers.
    And the file "main_gp.go" contains "//goplus:prop InRange[i, n] And[Le[0, i], Lt[i, n]]"
    And the file "main_gp.go" does not contain "prop {"
    # Unfolding reaches the hypotheses, so Lt[i, n] still prunes Nil.
    And the file "main_gp.go" contains:
      """
      panic("goplus: At: v with index n cannot be Nil")
      """

  Scenario: A named proposition that does not hold is refused, by name
    Given a Go+ file "main.gp":
      """
      package main

      type InRange[i nat, n nat] prop { And[Le[0, i], Lt[i, n]] }

      func At(0 i nat, 0 n nat, 0 p InRange[i, n]) int {
      	return 0
      }

      func Bad() int {
      	return At(5, 3, decide)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove InRange[5, 3] at this call to At"
    And the file "main_gp.go" does not exist

  Scenario: A named proposition may be asserted, and is audited by name
    Given a Go+ file "main.gp":
      """
      package main

      type InRange[i nat, n nat] prop { And[Le[0, i], Lt[i, n]] }

      func At(0 i nat, 0 n nat, 0 p InRange[i, n]) int {
      	return 0
      }

      func Asserted() int {
      	return At(5, 3, assume)
      }
      """
    When I run goplus with arguments "assumptions ."
    Then the exit code is 0
    And stdout contains "assumed InRange[5, 3] for p of At"

  Scenario: A named proposition may name another
    Given a Go+ file "main.gp":
      """
      package main

      type Pos[n nat] prop { Lt[0, n] }

      type Both[a nat, b nat] prop { And[Pos[a], Pos[b]] }

      func Need(0 a nat, 0 b nat, 0 p Both[a, b]) int {
      	return 0
      }

      func Ok() int {
      	return Need(1, 2, decide)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go

  Scenario: std/vec indexes by a plain number, and rejects an out-of-range one
    Given a module "example.com/demo" using the goplus standard library
    And a Go+ file "main.gp":
      """
      package main

      import "goforge.dev/goplus/std/vec"

      func Good(v vec.Vec[int, 3]) int {
      	return vec.AtIndex(2, 3, decide, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    # The bound erases: the call is an ordinary two-argument walk.
    And the file "main_gp.go" contains "vec.AtIndex(2, v)"
    Given a Go+ file "main.gp":
      """
      package main

      import "goforge.dev/goplus/std/vec"

      func Bad(v vec.Vec[int, 3]) int {
      	return vec.AtIndex(7, 3, decide, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove 7 < 3 at this call to AtIndex"
