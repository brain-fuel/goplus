Feature: Lowered function naming and visibility
  A generic method (R).M lowers to a package-level function named
  concat(R, M). The function is exported if and only if BOTH the receiver
  type and the method are exported — lowering never widens or narrows what
  other packages could already reach.

  Scenario Outline: Name and visibility of the lowered function
    Given a receiver type "<type>" and method "<method>"
    Then the lowered function name is "<name>"

    Examples:
      | type  | method | name     |
      | Stack | Map    | StackMap |
      | stack | Map    | stackMap |
      | Stack | map    | stackMap |
      | stack | map    | stackMap |
      | Tree  | Fold   | TreeFold |
      | ring  | rotate | ringRotate |

  Scenario: A //gpp:name override wins verbatim
    Given a receiver type "Stack" and method "Map" with name override "MapStack"
    Then the lowered function name is "MapStack"
