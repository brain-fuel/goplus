Feature: Generation outside a module
  Pass 1 does not lower a Go+ construct to Go. It lowers it to a SKELETON
  plus a carrier that resolution is contracted to consume — a match becomes
  `case nil:` heads whose bodies are `//goplus:pattern` comments, a pipeline
  becomes a `__gp_bare_` call. Resolution needs type information, so it
  needs a module. Without one it does not run, and the skeleton used to be
  written out as if it were the finished artifact: two `case nil:` arms in
  one switch, binders that were never bound, no return — invalid Go, from a
  command that exited 0.

  So the artifact is checked for the carriers before it is written. This is
  the same rule typed holes already followed, one step more general: what
  resolution was supposed to finish must not be mistaken for the finished
  thing.

  Scenario: A match outside a module is refused, not half-generated
    Given a Go+ file "main.gp":
      """
      package demo

      type Shape enum {
      	Circle(R float64)
      	Square(S float64)
      }

      func Area(s Shape) float64 {
      	match s {
      	case Circle(r):
      		return 3.14159 * r * r
      	case Square(w):
      		return w * w
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "a match cannot be generated here: this package has no module context"
    And the file "main_gp.go" does not exist

  Scenario: A pipeline outside a module is refused too
    Given a Go+ file "main.gp":
      """
      package demo

      func double(x int) int { return x * 2 }

      func Quad(x int) int {
      	return x |> double |> double
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "a pipeline cannot be generated here: this package has no module context"
    And the file "main_gp.go" does not exist

  # The refusal is about carriers, not about the absence of a module. Go+
  # that needs no resolution still generates outside one, which is what
  # makes a single .gp file usable as a script.
  Scenario: Go+ that needs no resolution still generates outside a module
    Given a Go+ file "main.gp":
      """
      package demo

      type Shape enum {
      	Circle(R float64)
      	Square(S float64)
      }

      func Name() string { return "shapes" }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    And the file "main_gp.go" contains "func Fold[R any](s Shape, cs ShapeCases[R]) R"

  Scenario: The same match generates once the directory is a module
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """
    And a Go+ file "main.gp":
      """
      package demo

      type Shape enum {
      	Circle(R float64)
      	Square(S float64)
      }

      func Area(s Shape) float64 {
      	match s {
      	case Circle(r):
      		return 3.14159 * r * r
      	case Square(w):
      		return w * w
      	}
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" is valid Go
    And the file "main_gp.go" contains "case Circle:"
    And the file "main_gp.go" does not contain "//goplus:pattern"
