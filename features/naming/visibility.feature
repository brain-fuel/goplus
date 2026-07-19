Feature: Lowered function naming and visibility
  A generic method (R).M lowers to a package-level function that keeps the
  METHOD'S OWN NAME — the name is already in the code — cased so the
  result is exported if and only if BOTH the receiver type and the method
  are exported (lowering never widens or narrows what other packages could
  already reach). When the name is shared inside the package, every
  collider falls back to the receiver-prefixed concat(R, M) form, the same
  discipline enum variant structs use.

  Scenario Outline: Name and visibility of the lowered function
    Given a receiver type "<type>" and method "<method>"
    Then the lowered function name is "<name>"
    And the prefixed lowered function name is "<prefixed>"

    Examples:
      | type  | method | name   | prefixed   |
      | Stack | Map    | Map    | StackMap   |
      | stack | Map    | map    | stackMap   |
      | Stack | map    | map    | stackMap   |
      | stack | map    | map    | stackMap   |
      | Tree  | Fold   | Fold   | TreeFold   |
      | ring  | rotate | rotate | ringRotate |
