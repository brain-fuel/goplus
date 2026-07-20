Feature: Refinement types
  A refinement is a statically proved predicate over a value and erases to its
  Go base type. Construction is accepted from constants or path facts only
  when the predicate is established; an unproved cast is a compile error.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: Refinements erase and prove constants and path-narrowed values
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Positive refine(value int) { value > 0 }

      func narrowed(n int) int {
      	if n > 0 {
      		return Positive(n)
      	}
      	return 0
      }

      func main() {
      	fmt.Println(Positive(3), narrowed(7), narrowed(-1))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "3 7 0"
    And the file "main_gp.go" contains:
      """
      //goplus:refinement "Positive" "value" "int" "value > 0"
      type Positive = int
      """

  Scenario: Unproved refinement construction is rejected
    Given a Go+ file "main.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      func unchecked(n int) Positive { return Positive(n) }
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove value > 0 for Positive(n)"

  Scenario: A false constant does not inhabit a refinement
    Given a Go+ file "main.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      var bad = Positive(0)
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove value > 0 for Positive(0)"

  Scenario: Refined parameters introduce facts and exported boundaries guard Go callers
    Given a Go+ file "positive.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      func EchoPositive(value Positive) Positive { return value }
      """
    And a file "main.go":
      """
      package main

      func main() { EchoPositive(0) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 1
    And stderr contains "goplus: EchoPositive: value violates refinement Positive"
    And the file "positive_gp.go" contains:
      """
      if !__goplus_refinement_Positive(value) {
      		panic("goplus: EchoPositive: value violates refinement Positive")
      	}
      """

  Scenario: Every refined result is proved
    Given a Go+ file "main.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      func invalid() Positive { return 0 }
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove value > 0 for value 0"

  Scenario: Refined function contracts survive alias erasure at calls
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Positive refine(value int) { value > 0 }

      func makePositive() Positive { return Positive(2) }
      func needPositive(value Positive) int { return value }

      func main() { fmt.Println(needPositive(makePositive())) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "2"

  Scenario: An ordinary value cannot bypass a refined function parameter
    Given a Go+ file "main.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      func needPositive(value Positive) int { return value }
      func unchecked(value int) int { return needPositive(value) }
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove value > 0 for argument 1 to needPositive"

  Scenario: Refined variables reject invalid initialization and reassignment
    Given a Go+ file "main.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      func invalid() {
      	var value Positive = Positive(1)
      	value = 0
      	_ = value
      }
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "cannot prove value > 0 for assignment to value"

  Scenario: A refinement without a valid zero value must be initialized
    Given a Go+ file "main.gp":
      """
      package main

      type Positive refine(value int) { value > 0 }

      var value Positive
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "zero value does not satisfy value > 0"
