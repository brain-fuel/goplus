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
    And the file "main_gp.go" does not contain "assume"

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
