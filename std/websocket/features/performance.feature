@benchmark
Feature: Comparative WebSocket performance
  Performance claims are executable contracts evaluated by benchgate.

  Scenario Outline: important primitives clear the reference implementation
    Given benchmark <benchmark>
    And reference implementation "github.com/gobwas/ws"
    Then Go+ WebSocket must be at least <ratio> times as fast
    And Go+ WebSocket must allocate no more bytes per operation

    Examples:
      | benchmark       | ratio |
      | ParseHeaderTiny | 2.0   |
      | ParseHeader64K  | 2.0   |
      | AppendHeader    | 2.0   |
      | Mask1K          | 2.0   |
