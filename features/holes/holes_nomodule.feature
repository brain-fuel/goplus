Feature: Typed holes without a module
  Computing a hole's goal needs type information, which needs a module. A
  hole must still never reach generated Go, so outside a module the hole is
  reported plainly and generation stops all the same.

  Scenario: The goal is unavailable, and nothing is written
    Given a Go+ file "main.gp":
      """
      package demo

      func Pick(xs []string) string {
        return ?choice
      }
      """
    When I run goplus with arguments "gen ."
    Then the exit code is 2
    And stderr contains "hole ?choice : this package has no module context"
    And the file "main_gp.go" does not exist
