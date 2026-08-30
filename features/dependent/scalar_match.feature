Feature: Literal patterns and scalar matches
  Stage C, second milestone, second half (v0.21.0 grammar; gap program
  C2b). A match may scrutinize an integer value: literal arms select by
  equality, guards compose, and a trailing binder or '_' arm catches
  the rest. Coverage is DECIDED: without a fallback arm the literals
  must cover 0..k-1 contiguously and a hypothesis in scope must prove
  scrutinee < k — the arithmetic decider's judgment, the same one that
  prunes enum variants. The generated Go still guards the boundary with
  a panic: hypotheses are static, plain Go callers are not. Literal
  patterns inside constructor arguments wait for C2c (bind and guard
  meanwhile).

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: A bounded nat matches by literals alone
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      func Name(n nat, 0 p Lt[n, 3]) string {
      	match n {
      	case 0:
      		return "zero"
      	case 1:
      		return "one"
      	case 2:
      		return "two"
      	}
      }

      func main() {
      	fmt.Println(Name(2, decide))
      	fmt.Println(Name(0, decide))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "two"
    And stdout contains "zero"
    And the file "main_gp.go" contains:
      """
      panic("goplus: match on n: value outside the proven range")
      """

  Scenario: Without the hypothesis, literal coverage is refused by name
    Given a Go+ file "main.gp":
      """
      package main

      func Name(n nat) string {
      	match n {
      	case 0:
      		return "zero"
      	case 1:
      		return "one"
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "no hypothesis in scope proves n < 2"

  Scenario: A binder arm makes any scalar match exhaustive, and guards compose
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      func Describe(n nat) string {
      	match n {
      	case 0:
      		return "none"
      	case 1 if n > 0:
      		return "single"
      	case k:
      		return fmt.Sprintf("many (%d)", k)
      	}
      }

      func main() {
      	fmt.Println(Describe(0))
      	fmt.Println(Describe(1))
      	fmt.Println(Describe(9))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "none"
    And stdout contains "single"
    And stdout contains "many (9)"

  Scenario: A duplicate literal is unreachable
    Given a Go+ file "main.gp":
      """
      package main

      func Pick(n nat) int {
      	match n {
      	case 0:
      		return 1
      	case 0:
      		return 2
      	case _:
      		return 3
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "unreachable match arm: 0"

  Scenario: Literals inside constructor arguments point at bind-and-guard
    Given a Go+ file "main.gp":
      """
      package main

      type Box enum {
      	Full(v int)
      	Empty()
      }

      func IsZero(b Box) bool {
      	match b {
      	case Full(0):
      		return true
      	case Full(v):
      		return v == 0
      	case Empty():
      		return false
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "bind and guard instead"
