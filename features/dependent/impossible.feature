Feature: Explicit impossible arms
  Stage C, first milestone (v0.19.0 grammar; gap program C1a). Pruning
  has been inferred since v0.7.0 — an arm whose variant clashes with the
  scrutinee's indices cannot be written at all. `impossible` makes the
  author's side of that statable: the arm asserts it is ruled out, the
  checker verifies the assertion against index clash under the
  hypotheses in scope, and the arm is dropped — the generated Go is the
  omitted-arm form, boundary guards included. A reachable arm marked
  impossible is an error naming what failed.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: A ruled-out arm may be asserted, and emits nothing
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func First[T any](0 n nat, v Vec[T, n+1]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	case Nil():
      		impossible
      	}
      }

      func main() {
      	fmt.Println(First(Cons(7, Nil[int]())))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "7"
    And the file "main_gp.go" contains:
      """
      panic("goplus: First: v with index n+1 cannot be Nil")
      """

  Scenario: A reachable arm marked impossible is refused, naming the index
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Sum(0 n nat, v Vec[int, n]) int {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	case Nil():
      		impossible
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "this arm is not impossible"
    And stderr contains "does not rule out Nil"
    And the file "main_gp.go" does not exist

  Scenario: A plain enum cannot host an impossible arm at all
    Given a Go+ file "main.gp":
      """
      package main

      type Color enum {
      	Red()
      	Blue()
      }

      func Pick(c Color) int {
      	match c {
      	case Red():
      		return 1
      	case Blue():
      		impossible
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "this arm is not impossible: Blue() can match a value of type Color"

  Scenario: A wildcard cannot be impossible
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func First[T any](0 n nat, v Vec[T, n+1]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	case _:
      		impossible
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "'case _:' cannot be impossible"

  Scenario: A proposition in scope discharges an impossible arm
    Given a Go+ file "main.gp":
      """
      package main

      type Vec[T any, n nat] enum {
      	Nil() Vec[T, 0]
      	Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func At[T any](0 i nat, 0 n nat, 0 p Lt[i, n], v Vec[T, n]) T {
      	match v {
      	case Cons(h, t):
      		_ = t
      		return h
      	case Nil():
      		impossible
      	}
      }

      func Ok(v Vec[int, 3]) int {
      	return At(1, 3, decide, v)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
