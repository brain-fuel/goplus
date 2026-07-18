Feature: Flow composes with enums and match
  Pipelines, partials, and composition are one language with enums and
  matching: flows appear inside match arms (both lowering modes), as match
  subjects, and feed constructors.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: Pipelines inside match arms feed constructors
    Given a G++ file "main.gpp":
      """
      package main

      import "fmt"

      type Option[T any] enum {
      	Some(value T)
      	None
      }

      func double(n int) int { return n * 2 }

      func (o Option[T]) MapDouble() Option[int] {
      	match o {
      	case Some(v):
      		return any(v).(int) |> double |> Some
      	case None:
      		return None
      	}
      }

      func main() {
      	var o Option[int] = Some(21)
      	fmt.Println(o.MapDouble())
      }
      """
    When I run gpp with arguments "run ."
    Then the exit code is 0
    And stdout contains "{42}"

  Scenario: A pipeline can be a match subject
    Given a G++ file "main.gpp":
      """
      package main

      import "fmt"

      type Option[T any] enum {
      	Some(value T)
      	None
      }

      func classify(n int) Option[int] {
      	if n > 0 {
      		return Some(n)
      	}
      	return None
      }

      func double(n int) int { return n * 2 }

      func main() {
      	match 21 |> double |> classify {
      	case Some(v):
      		fmt.Println("some", v)
      	case None:
      		fmt.Println("none")
      	}
      }
      """
    When I run gpp with arguments "run ."
    Then the exit code is 0
    And stdout contains "some 42"

  Scenario: Flows inside nested-mode match arms
    Given a G++ file "main.gpp":
      """
      package main

      import "fmt"

      type Expr enum {
      	Lit(v int)
      	Add(l, r Expr)
      }

      func double(n int) int { return n * 2 }

      func eval(e Expr) int {
      	match e {
      	case Add(Lit(a), Lit(b)):
      		return a + b |> double
      	case Add(l, r):
      		return eval(l) + eval(r)
      	case Lit(v):
      		return v
      	}
      }

      func main() {
      	fmt.Println(eval(Add(Lit(20), Lit(1))))
      }
      """
    When I run gpp with arguments "run ."
    Then the exit code is 0
    And stdout contains "42"

  Scenario: Cross-package flows use dep methods and constructors from markers
    Given a file "dep/go.mod":
      """
      module example.com/dep

      go 1.24
      """
    And a G++ file "dep/lib/option.gpp":
      """
      package lib

      type Option[T any] enum {
      	Some(value T)
      	None
      }

      func (o Option[T]) UnwrapOr(fb T) T {
      	match o {
      	case Some(v):
      		return v
      	case None:
      		return fb
      	}
      }
      """
    And I run gpp in "dep" with arguments "gen ./..."
    And the file "dep/lib/option.gpp" is deleted
    And a file "app/go.mod":
      """
      module example.com/app

      go 1.24

      require example.com/dep v0.0.0

      replace example.com/dep => ../dep
      """
    And a G++ file "app/main.gpp":
      """
      package main

      import (
      	"fmt"

      	"example.com/dep/lib"
      )

      func double(n int) int { return n * 2 }

      func main() {
      	got := 21 |> lib.Some |> .UnwrapOr(0) |> double
      	fmt.Println(got)
      }
      """
    When I run gpp in "app" with arguments "run ."
    Then the exit code is 0
    And stdout contains "42"
