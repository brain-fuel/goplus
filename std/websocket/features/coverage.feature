@coverage
Feature: Complete handwritten implementation coverage
  Generated Go+ algebra is checked by exhaustive fold and typestate tests;
  every reachable statement in the handwritten wire implementation must be
  exercised by deterministic unit, integration, property, or fault tests.

  Scenario: every handwritten statement is exercised
    Given the Go coverage profile for the WebSocket package
    Then handwritten statement coverage is 100 percent
