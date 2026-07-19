Feature: G++ test sources
  foo_test.gpp emits foo_gpp_test.go — still a _test.go file to the go
  tool — so tests are written in full G++: constructors, matches,
  pipelines, and derived generators all work in test code.

  Scenario: A test written in G++ runs under gpp test
    Given a module "example.com/demo" using rapid for law tests
    And a G++ file "main.gpp":
      """
      package main

      type Color enum {
      	Red
      	Blue(depth int)
      }

      func main() {}
      """
    And a G++ file "main_test.gpp":
      """
      package main

      import "testing"

      func TestMatchInGppTest(t *testing.T) {
      	var c Color = Blue(3)
      	match c {
      	case Blue(d):
      		if d != 3 {
      			t.Fatal("wrong depth")
      		}
      	case Red():
      		t.Fatal("wrong variant")
      	}
      }
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 0
    And the file "main_gpp_test.go" contains:
      """
      func TestMatchInGppTest(t *testing.T) {
      """
    When I run gpp with arguments "test ."
    Then the exit code is 0
