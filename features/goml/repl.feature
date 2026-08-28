Feature: the goml REPL
  goml has no interpreter, so the REPL compiles and runs: every input
  transpiles the accumulated session, generates Go through the ordinary
  pipeline, and runs it. Declarations are retained; expression results
  are not, so an effectful expression runs exactly once.

  Scenario: Declaring then evaluating
    When I run goml with arguments "repl" and input:
      """
      let Double (n : Int) : Int := n * 2
      Double 21
      :quit
      """
    Then the exit code is 0
    And stdout contains "42"

  Scenario: Value bindings compose across inputs
    When I run goml with arguments "repl" and input:
      """
      let Port := 8080
      let Next : Int := Port + 1
      Next
      :quit
      """
    Then the exit code is 0
    And stdout contains "8081"

  Scenario: A multi-line declaration is submitted by a blank line
    When I run goml with arguments "repl" and input:
      """
      type Color :=
        | Red
        | Green

      let Name (c : Color) : String :=
        match c with
        | Red => "red"
        | Green => "green"

      Name Green
      :quit
      """
    Then the exit code is 0
    And stdout contains "green"

  Scenario: A failed input leaves the session intact
    When I run goml with arguments "repl" and input:
      """
      let A := 1
      let Bad : Int := "nope"
      A + 1
      :quit
      """
    Then the exit code is 0
    And stderr contains "cannot use"
    And stdout contains "2"

  Scenario: Breaking an earlier binding names it
    When I run goml with arguments "repl" and input:
      """
      let A := 1
      let Uses : Int := A + 1
      let A := "text"
      :quit
      """
    Then the exit code is 0
    And stderr contains "in Uses (defined earlier)"

  Scenario: The last result is available as it
    When I run goml with arguments "repl" and input:
      """
      21 * 2
      it + 1
      :quit
      """
    Then the exit code is 0
    And stdout contains "43"

  Scenario: help states the replay caveat
    When I run goml with arguments "repl" and input:
      """
      :help
      :quit
      """
    Then the exit code is 0
    And stdout contains "re-execute on every evaluation"

  Scenario: :type reports a declared binding's own signature
    When I run goml with arguments "repl" and input:
      """
      let Twice (n : Int) : Int := n * 2
      :type Twice
      :quit
      """
    Then the exit code is 0
    And stdout contains "Twice : (n : Int) -> Int"

  Scenario: A declaration with a hole prints its goal and is not retained
    When I run goml with arguments "repl" and input:
      """
      let Half (n : Int) : Int := ?impl
      :holes
      :quit
      """
    Then the exit code is 0
    And stderr contains "hole ?impl : Int"
    And stderr contains "not retained"
    And stdout contains "?impl : Int"
