Feature: Option pipelines
  When the value flowing through |> is an Option[T] — the single-track
  sibling of the Result railway (gap program T1.5) — stages lift by
  shape, first match wins: dot segments receive the raw Option; a stage
  accepting the Option stays a direct call; T→Option[U] binds;
  T→(value, ok) adapts through option.Of (Go's comma-ok shape); T→U
  maps. None bypasses every lifted stage. Option has no error track and
  no Tee: a stage returning nothing is a hard error. A constructor
  literal head (option.Some(16)) is on the pipeline the same as an
  Option-typed value.

  Background:
    Given a module "example.com/demo" using the goplus standard library

  Scenario: The canonical Option chain — Bind, comma-ok adapt, Map, dot segment
    Given a Go+ file "main.gp":
      """
      package main

      import (
      	"fmt"
      	"strconv"

      	"goforge.dev/goplus/std/option"
      )

      func half(n int) option.Option[int] {
      	if n%2 != 0 {
      		return option.None[int]()
      	}
      	return option.Some(n / 2)
      }

      func lookup(k string) (int, bool) {
      	m := map[string]int{"a": 8}
      	v, ok := m[k]
      	return v, ok
      }

      func main() {
      	n := option.Some(16) |> half |> half |> strconv.Itoa |> .UnwrapOr("none")
      	fmt.Println(n)
      	m := option.Some("a") |> lookup |> half |> .UnwrapOr(-1)
      	fmt.Println(m)
      	gone := option.Some(3) |> half |> half |> .UnwrapOr(-1)
      	fmt.Println(gone)
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "4"
    And stdout contains "-1"
    And the file "main_gp.go" contains "option.Bind"
    And the file "main_gp.go" contains "option.Map"
    And the file "main_gp.go" contains "return option.Of(lookup(__gp_p))"

  Scenario: A stage returning nothing cannot lift — Option has no Tee
    Given a Go+ file "main.gp":
      """
      package main

      import "goforge.dev/goplus/std/option"

      func drop(n int) {}

      func run(o option.Option[int]) option.Option[int] {
      	return o |> drop
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "Option has no Tee"

  Scenario: A stage accepting the Option itself stays a direct call
    Given a Go+ file "main.gp":
      """
      package main

      import (
      	"fmt"

      	"goforge.dev/goplus/std/option"
      )

      func describe(o option.Option[int]) string {
      	if option.IsSome(o) {
      		return "some"
      	}
      	return "none"
      }

      func main() {
      	fmt.Println(option.Some(1) |> describe)
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "some"
    And the file "main_gp.go" contains "describe(option.Some[int]{Value: 1})"
    And the file "main_gp.go" does not contain "option.Bind(describe"

  Scenario: Stage extra arguments close over and evaluate on the Some path only
    Given a Go+ file "main.gp":
      """
      package main

      import (
      	"fmt"

      	"goforge.dev/goplus/std/option"
      )

      func add(a, b int) int { return a + b }

      func main() {
      	fmt.Println(option.Some(2) |> add(40) |> .UnwrapOr(0))
      	fmt.Println(option.None[int]() |> add(40) |> .UnwrapOr(-7))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "42"
    And stdout contains "-7"
