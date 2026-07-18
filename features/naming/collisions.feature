Feature: Name collisions are compile errors
  Generated names must be stable, deterministic public API. Any collision —
  with an authored declaration, with another generated name, or via the
  visibility rule's case folding — is an error naming both origins, fixable
  with an explicit //gpp:name directive.

  Scenario: Generated name collides with an authored function
    Given a G++ file "stack.gpp":
      """
      package stack

      type Stack[T any] struct{ items []T }

      func StackMap() {}

      func (s Stack[T]) Map[U any](f func(T) U) Stack[U] {
      	return Stack[U]{}
      }
      """
    When I compute lowered names
    Then name generation fails with an error containing "generated name StackMap"
    And name generation fails with an error containing "collides"
    And name generation fails with an error containing "//gpp:name"

  Scenario: Two generated names collide through concatenation ambiguity
    Given a G++ file "ambi.gpp":
      """
      package ambi

      type AB[T any] struct{}
      type A[T any] struct{}

      func (x AB[T]) C[U any]()  {}
      func (x A[T]) BC[U any]()  {}
      """
    When I compute lowered names
    Then name generation fails with an error containing "generated name ABC"
    And name generation fails with an error containing "generated function"

  Scenario: Case folding makes unexported twins collide
    Given a G++ file "fold.gpp":
      """
      package fold

      type Ring[T any] struct{}
      type ring[T any] struct{}

      func (r ring[T]) Rotate[U any]() {}
      func (r Ring[T]) rotate[U any]() {}
      """
    When I compute lowered names
    Then name generation fails with an error containing "generated name ringRotate"

  Scenario: //gpp:name resolves a collision
    Given a G++ file "fixed.gpp":
      """
      package fixed

      type Stack[T any] struct{ items []T }

      func StackMap() {}

      //gpp:name MapStack
      func (s Stack[T]) Map[U any](f func(T) U) Stack[U] {
      	return Stack[U]{}
      }
      """
    When I compute lowered names
    Then the lowered names are "MapStack"

  Scenario: An invalid //gpp:name is rejected
    Given a G++ file "bad.gpp":
      """
      package bad

      type Stack[T any] struct{}

      //gpp:name not a name
      func (s Stack[T]) Map[U any]() {}
      """
    When I compute lowered names
    Then name generation fails with an error containing "not a valid Go identifier"
