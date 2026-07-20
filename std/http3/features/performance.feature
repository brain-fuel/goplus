@benchmark
Feature: Comparative HTTP 3 performance
  Performance claims are executable contracts evaluated against quic-go.

  Scenario Outline: important field-section encoders clear the reference
    Given benchmark <benchmark>
    And reference implementation "github.com/quic-go/quic-go"
    Then Go+ HTTP 3 must be at least <ratio> times as fast
    And Go+ HTTP 3 must allocate no more bytes per operation

    Examples:
      | benchmark             | ratio |
      | NativeStackRoundTrip  | 2.0   |
      | DecodeRegularFields   | 2.0   |
      | EncodeWebSocketFields | 2.0   |
      | EncodeRegularFields   | 2.0   |
      | EncodeFrameHeader     | 2.0   |
