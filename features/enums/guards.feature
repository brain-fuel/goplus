Feature: Match guards
  Stage C, second milestone, first half (v0.20.0 grammar; gap program
  C2a). `case Cons(h, t) if h > n:` runs its arm only when the guard
  holds; a false guard falls through to the next arm. Guards contribute
  NOTHING to exhaustiveness — a guard may be false, so the match still
  needs unguarded coverage — and a guarded match lowers through the
  goto chain, where the guard is one more check evaluated after the
  pattern's bindings. A wildcard cannot be guarded, a multi-pattern arm
  cannot be guarded, and an impossible arm cannot be guarded.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: Guards select by value and fall through in order
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Shape enum {
      	Circle(r int)
      	Rect(w int, h int)
      }

      func Classify(s Shape) string {
      	match s {
      	case Circle(r) if r > 10:
      		return "big circle"
      	case Circle(r):
      		return fmt.Sprintf("circle %d", r)
      	case Rect(w, h) if w == h:
      		return "square"
      	case Rect(w, h):
      		_, _ = w, h
      		return "rect"
      	}
      }

      func main() {
      	fmt.Println(Classify(Circle(12)))
      	fmt.Println(Classify(Circle(3)))
      	fmt.Println(Classify(Rect(4, 4)))
      	fmt.Println(Classify(Rect(4, 5)))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "big circle"
    And stdout contains "circle 3"
    And stdout contains "square"
    And stdout contains "rect"

  Scenario: A guarded arm covers nothing — exhaustiveness still demands the variant
    Given a Go+ file "main.gp":
      """
      package main

      type Color enum {
      	Red()
      	Blue()
      }

      func Pick(c Color) int {
      	match c {
      	case Red() if true:
      		return 1
      	case Blue():
      		return 2
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "non-exhaustive match on Color: missing Red"

  Scenario: A wildcard cannot be guarded
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
      	case _ if true:
      		return 2
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "'case _:' cannot take a guard"

  Scenario: A guard sees the pattern's binders, even when the body does not use them
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Box enum {
      	Full(v int)
      	Empty()
      }

      func Sign(b Box) string {
      	match b {
      	case Full(v) if v < 0:
      		return "negative"
      	case Full(v):
      		_ = v
      		return "non-negative"
      	case Empty():
      		return "empty"
      	}
      }

      func main() {
      	fmt.Println(Sign(Full(-3)))
      	fmt.Println(Sign(Full(3)))
      	fmt.Println(Sign(Empty()))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "negative"
    And stdout contains "non-negative"
    And stdout contains "empty"
