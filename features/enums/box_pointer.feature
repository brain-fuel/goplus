Feature: Pointer-boxed enums
  `//goplus:box pointer` stores an enum's variants behind pointers: the sealed
  marker methods are generated on pointer receivers, so only `*Variant`
  satisfies the interface and construction is `&Variant{…}`. This matches the
  ubiquitous Go pattern of pointer-boxed sum types — large nodes that are
  mutated in place rather than copied (ASTs, IRs). A `match` over a
  pointer-boxed enum lowers to `case *Variant:` heads and binds the pointer, so
  exhaustive match — and in-place mutation through the bound scrutinee — work on
  such enums. Pointer-boxed enums derive no value-semantic Fold/Equal/Traversal
  (they are consumed via match/type-switch); a bare variant pattern binds the
  pointer without field binders.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: Only the pointer satisfies the interface, and match binds it for mutation
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      //goplus:box pointer
      type Expr enum {
      	Binary(Left int, Right int)
      	Lit(Value int)
      }

      func eval(e Expr) int {
      	match e {
      	case b := Binary:
      		b.Left = b.Left + 10 // mutate through the bound pointer
      		return b.Left + b.Right
      	case l := Lit:
      		return l.Value
      	}
      }

      func main() {
      	fmt.Println(eval(&Binary{Left: 1, Right: 2}))
      	fmt.Println(eval(&Lit{Value: 7}))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains:
      """
      13
      7
      """

  Scenario: The marker methods are generated on pointer receivers
    Given a Go+ file "main.gp":
      """
      package main

      //goplus:box pointer
      type Expr enum {
      	Binary(Left int, Right int)
      	Lit(Value int)
      }

      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" contains "func (*Binary) isExpr() {}"
    And the file "main_gp.go" contains "func (*Lit) isExpr() {}"
    And the file "main_gp.go" contains "type Expr interface{ isExpr() }"

  Scenario: A non-exhaustive match on a pointer-boxed enum is a compile error naming the variant
    Given a Go+ file "main.gp":
      """
      package main

      //goplus:box pointer
      type Expr enum {
      	Binary(Left int, Right int)
      	Lit(Value int)
      }

      func eval(e Expr) int {
      	match e {
      	case b := Binary:
      		return b.Left
      	}
      }

      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 1
    And stderr contains "non-exhaustive match on Expr: missing Lit"

  Scenario: A wildcard arm opts out of exhaustiveness
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      //goplus:box pointer
      type Expr enum {
      	Binary(Left int, Right int)
      	Lit(Value int)
      }

      func kind(e Expr) string {
      	match e {
      	case l := Lit:
      		return fmt.Sprintf("lit %d", l.Value)
      	case _:
      		return "other"
      	}
      }

      func main() {
      	fmt.Println(kind(&Lit{Value: 3}))
      	fmt.Println(kind(&Binary{Left: 1, Right: 2}))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains:
      """
      lit 3
      other
      """

  Scenario: A pointer-boxed enum derives no value-semantic fold
    Given a Go+ file "main.gp":
      """
      package main

      //goplus:box pointer
      type Expr enum {
      	Binary(Left int, Right int)
      	Lit(Value int)
      }

      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 0
    And the file "main_gp.go" does not contain "func Fold"
    And the file "main_gp.go" does not contain "ExprCases"
