Feature: RFC 7692 permessage-deflate
  Compression negotiation is explicit, directional, bounded, and rejects
  parameters that were not offered. Message decompression never bypasses the
  configured application-message limit.

  Scenario: a client advertises both window limits
    Given compression options with client window 9 and server window 10
    When I build the permessage-deflate offer
    Then the extension offer is "permessage-deflate; server_no_context_takeover; client_no_context_takeover; server_max_window_bits=10; client_max_window_bits=9"

  Scenario: a server selects windows no larger than the offer
    Given compression options with client window 8 and server window 9
    When I negotiate offer "permessage-deflate; server_no_context_takeover; client_no_context_takeover; server_max_window_bits=10; client_max_window_bits=12"
    Then the server write window is 9
    And the server read window is 8

  Scenario: an unoffered response parameter is rejected
    Given compression options with client window 0 and server window 0
    When I validate response "permessage-deflate; server_no_context_takeover; client_no_context_takeover; client_max_window_bits=9"
    Then extension negotiation fails

  Scenario: decompressed size is bounded
    Given 4096 repeated bytes compressed with window 8
    When I decompress with a 4095 byte limit
    Then decompression fails with message too large
