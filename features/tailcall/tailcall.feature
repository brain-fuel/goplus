Feature: Native tail calls
  `recur(nextArgs...)` is a contextual statement intrinsic. In the final
  position of a function or tail branch it rebinds the function parameters
  simultaneously and starts the body again in the same Go stack frame.
  Method receivers remain fixed; every ordinary parameter is explicit.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: Tail recursion lowers to a loop and runs with constant stack
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      tail func sumTo(n, acc uint64) uint64 {
        if n == 0 {
          return acc
        }
        recur(n-1, acc+n)
      }

      func main() { fmt.Println(sumTo(1_000_000, 0)) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "500000500000"
    And the file "main_gp.go" contains:
      """
      for {
      """
    And the file "main_gp.go" contains:
      """
      n, acc = n-1, acc+n
      """
    And the file "main_gp.go" contains:
      """
      continue __goplus_recur
      """
    And the file "main_gp.go" does not contain "recur("

  Scenario: Rebinding is simultaneous and arguments evaluate left to right
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      func note(log *[]int, value int) int {
        *log = append(*log, value)
        return value
      }

      tail func rotate(a, b, turns int, log *[]int) (int, int) {
        if turns == 0 {
          return a, b
        }
        recur(note(log, b), note(log, a), turns-1, log)
      }

      func main() {
        log := []int{}
        a, b := rotate(1, 2, 1, &log)
        fmt.Println(a, b, log)
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "2 1 [2 1]"

  Scenario: Tail calls work in lowered match arms
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type List enum {
        Nil
        Cons(value int, rest List)
      }

      tail func sum(xs List, acc int) int {
        match xs {
        case Nil:
          return acc
        case Cons(x, rest):
          recur(rest, acc+x)
        }
      }

      func main() { fmt.Println(sum(Cons(1, Cons(2, Cons(3, Nil))), 0)) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "6"
    And the file "main_gp.go" does not contain "recur("

  Scenario: Tail calls survive nested-pattern chain lowering
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type List enum {
        Nil
        Cons(value int, rest List)
      }

      tail func pairSum(xs List, acc int) int {
        match xs {
        case Nil:
          return acc
        case Cons(x, Nil):
          return acc+x
        case Cons(x, Cons(y, rest)):
          recur(rest, acc+x+y)
        }
      }

      func main() {
        xs := Cons(1, Cons(2, Cons(3, Cons(4, Nil))))
        fmt.Println(pairSum(xs, 0))
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "10"
    And the file "main_gp.go" does not contain "recur("

  Scenario: A non-tail recur is rejected
    Given a Go+ file "main.gp":
      """
      package main

      tail func bad(n int) int {
        recur(n-1)
        return n
      }

      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "recur must be a final statement of the function or a tail branch"

  Scenario: Recur arity must match the parameter state
    Given a Go+ file "main.gp":
      """
      package main

      tail func bad(a, b int) int { recur(a) }
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "recur has 1 arguments, want 2 function parameters"

  Scenario: Total tail recursion is termination checked and reconstructible
    Given a Go+ file "count/count.gp":
      """
      package count

      total func Down(n, acc nat) nat {
        if n == 0 {
          return acc
        }
        recur(n-1, acc+1)
      }
      """
    And a Go+ file "main.gp":
      """
      package main

      import (
        "fmt"
        "example.com/demo/count"
      )

      total func Result(n nat) nat { return count.Down(n, 0) }

      func main() { fmt.Println(Result(1000000)) }
      """
    When I run goplus with arguments "gen ./..."
    Then the exit code is 0
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "1000000"

  Scenario: Total recur must structurally decrease
    Given a Go+ file "main.gp":
      """
      package main

      total func Bad(n nat) nat { recur(n+1) }
      func main() {}
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "total function Bad does not terminate: this recursive call shrinks no argument"

  Scenario: Ordinary Go symbols named recur remain untouched
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      func recur(n int) int { return n + 1 }
      func ordinary(n int) int {
        recur(n)
        return recur(n)
      }
      func main() { fmt.Println(ordinary(4)) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "5"
    And the file "main_gp.go" contains:
      """
      return recur(n)
      """

  Scenario: Loop semantics retain the receiver, named results, and one defer stack
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Counter int

      tail func (c Counter) add(n int, defers *int) (out int) {
        defer func() { *defers++ }()
        out++
        if n == 0 {
          return int(c) + out
        }
        recur(n-1, defers)
      }

      func main() {
        defers := 0
        fmt.Println(Counter(10).add(3, &defers), defers)
      }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "14 4"

  Scenario: A lowered generic-method receiver remains fixed
    Given a Go+ file "main.gp":
      """
      package main

      import "fmt"

      type Box[T any] struct{ Value T }

      tail func (b Box[T]) Repeat[U any](n int, value U) U {
        if n == 0 {
          return value
        }
        recur(n-1, value)
      }

      func main() { fmt.Println(Box[int]{Value: 1}.Repeat(1000000, "done")) }
      """
    When I run goplus with arguments "run ."
    Then the exit code is 0
    And stdout contains "done"
    And the file "main_gp.go" contains:
      """
      n, value = n-1, value
      """
    And the file "main_gp.go" does not contain "b, n, value ="
