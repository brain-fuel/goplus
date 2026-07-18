Feature: Diagnostics point at .gpp source
  gpp type-checks the final emitted Go before writing anything (the
  backstop). Errors — whether in G++ constructs or plain Go regions — are
  reported at .gpp positions, and a failing gen writes no files.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: A type error inside a generic method body maps to its .gpp line
    Given a G++ file "main.gpp":
      """
      package main

      type Stack[T any] struct{ items []T }

      func (s Stack[T]) Map[U any](f func(T) U) Stack[U] {
      	var wrong int = "not an int"
      	_ = wrong
      	return Stack[U]{}
      }

      func main() {}
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gpp:6:"
    And stderr contains "cannot use"
    And the file "main_gpp.go" does not exist

  Scenario: A type error in a plain-Go passthrough region maps exactly
    Given a G++ file "main.gpp":
      """
      package main

      import "fmt"

      func main() {
      	fmt.Println(undefinedThing)
      }
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gpp:6:14: undefined: undefinedThing"
    And the file "main_gpp.go" does not exist

  Scenario: Misusing a generic method call surfaces at the call site
    Given a G++ file "main.gpp":
      """
      package main

      type Stack[T any] struct{ items []T }

      func (s Stack[T]) Map[U any](f func(T) U) Stack[U] {
      	return Stack[U]{}
      }

      func main() {
      	s := Stack[int]{}
      	s.Map(42)
      }
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gpp:11:"

  Scenario: A misspelled method is an ordinary undefined-selector error
    Given a G++ file "main.gpp":
      """
      package main

      type Stack[T any] struct{ items []T }

      func (s Stack[T]) Map[U any](f func(T) U) Stack[U] {
      	return Stack[U]{}
      }

      func main() {
      	s := Stack[int]{}
      	s.Mapp(func(x int) int { return x })
      }
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gpp:11:"
    And stderr contains "Mapp"

  Scenario: Failed inference reads like a Go inference error
    Given a G++ file "main.gpp":
      """
      package main

      type Stack[T any] struct{ items []T }

      func (s Stack[T]) Empty[U any]() Stack[U] {
      	return Stack[U]{}
      }

      func main() {
      	s := Stack[int]{}
      	_ = s.Empty()
      }
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "main.gpp:11:"
    And stderr contains "cannot infer U"
