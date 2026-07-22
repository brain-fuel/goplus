Feature: Transparent single-variant enums
  `//goplus:repr transparent` gives a monomorphic single-variant enum a
  concrete Go alias representation. Dependent indices and enum markers remain
  available to Go+ and cross-package consumers, while ordinary Go calls avoid
  interface boxing. Multi-variant and existential enums are rejected because
  they require a runtime sum representation.

  Background:
    Given a file "go.mod":
      """
      module example.com/transparent

      go 1.26.0
      """

  Scenario: Indexed zero-cost wrapper retains match semantics
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      //goplus:derive off
      //goplus:repr transparent
      type Handle[n nat] enum { handleValue(ID int) Handle[n] }

      func New(id nat) Handle[id] { return handleValue(int(id)) }

      func value(0 n nat, handle Handle[n]) int {
      	match handle { case handleValue(id): return id }
      }

      func main() { fmt.Println(value(New(7))) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "7"
    And the file "main_gp.go" contains "type Handle = handleValue"

  Scenario: Sum types cannot request transparent representation
    Given a Go+ file "main.gp":
      """
      package main

      //goplus:repr transparent
      type Choice enum { Left(Value int); Right(Value int) }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "//goplus:repr transparent requires exactly one variant"
