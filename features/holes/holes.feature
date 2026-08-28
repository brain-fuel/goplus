Feature: Typed holes
  `?name` stands where code is not written yet. Generation stops and reports
  the hole's goal: the type it must produce — un-erased where the context is
  dependent — and the bindings available to produce it with. A hole is never
  written to generated Go, so a committed artifact is always complete.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: A hole reports the type its context expects
    Given a Go+ file "main.gp":
      """
      package demo

      func Pick(xs []string) string {
        return ?choice
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gp:4:10: hole ?choice : string"
    And stderr contains:
      """
        in scope:
          xs : []string
      """
    And the file "main_gp.go" does not exist

  Scenario: A hole in argument position takes the callee's parameter type
    Given a Go+ file "main.gp":
      """
      package demo

      func Join(sep string, parts []string) string { return sep }

      func Use() string {
        return Join(",", ?parts)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?parts : []string"
    And the file "main_gp.go" does not exist

  Scenario: A dependent hole reports the un-erased goal and the erased indices in scope
    Given a Go+ file "main.gp":
      """
      package demo

      type Vec[T any, n nat] enum {
        Nil() Vec[T, 0]
        Cons(head T, tail Vec[T, n]) Vec[T, n+1]
      }

      func Rest[T any](0 n nat, v Vec[T, n+1]) Vec[T, n] {
        return ?rest
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gp:9:10: hole ?rest : Vec[T, n]"
    And stderr contains:
      """
        erased: Vec[T]
      """
    And stderr contains:
      """
          n : nat (erased, quantity 0)
          v : Vec[T, n+1]
      """
    And the file "main_gp.go" does not exist

  Scenario: A hole whose context fixes no type says so and names the fix
    Given a Go+ file "main.gp":
      """
      package demo

      func Untyped() {
        x := ?mystery
        _ = x
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?mystery : cannot infer a type from this context"
    And stderr contains "hint: annotate the binding this hole initializes"
    And the file "main_gp.go" does not exist

  Scenario: A hole in function position reports the signature the call demands
    Given a Go+ file "main.gp":
      """
      package demo

      func Called(xs []int) int {
        return ?fn(xs, 3)
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?fn : func([]int, int) int"
    And the file "main_gp.go" does not exist

  Scenario: Every hole in a file is reported
    Given a Go+ file "main.gp":
      """
      package demo

      func Two(a int, b string) (int, string) {
        return ?first, ?second
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?first : int"
    And stderr contains "hole ?second : string"
    And the file "main_gp.go" does not exist

  Scenario: A hole leaves a previously generated output untouched
    Given a Go+ file "main.gp":
      """
      package demo

      func Ready(n int) int { return n + 1 }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And I record the generated files
    Given a Go+ file "main.gp":
      """
      package demo

      func Ready(n int) int { return n + 1 }

      func NotReady(n int) int { return ?later }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?later : int"
    And the generated files are unchanged

  Scenario: A package-level value may hold a hole
    Given a Go+ file "main.gp":
      """
      package demo

      var Greeting string = ?message
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?message : string"
    And the file "main_gp.go" does not exist

  Scenario: A spaced question mark is not a hole
    Given a Go+ file "main.gp":
      """
      package demo

      func Bad() int {
        return ? oops
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "illegal character"
    And the file "main_gp.go" does not exist

  Scenario: Postfix try still claims an adjacent question mark
    Given a Go+ file "main.gp":
      """
      package demo

      import "strconv"

      func Parse(s string) (int, error) {
        n := strconv.Atoi(s)?
        return n + 1, nil
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" contains:
      """
      n, __gp_err0 := strconv.Atoi(s)
      """
