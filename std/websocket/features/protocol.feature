Feature: RFC 6455 wire protocol
  The low-level codec accepts exactly canonical RFC 6455 frames and preserves
  payload bytes across masking, fragmentation, and control-frame interleaving.

  Scenario: RFC opening-handshake example
    When I compute the accept key for "dGhlIHNhbXBsZSBub25jZQ=="
    Then the accept key is "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

  Scenario Outline: canonical frame lengths round trip
    Given a final binary frame of length <length> received by a client
    When I encode and parse the header
    Then parsing succeeds and consumes <bytes> header bytes
    And the parsed payload length is <length>

    Examples:
      | length | bytes |
      | 0      | 2     |
      | 125    | 2     |
      | 126    | 4     |
      | 65535  | 4     |
      | 65536  | 10    |

  Scenario: a server rejects an unmasked client frame
    Given the wire header bytes "8100"
    When a server parses the header
    Then parsing fails with "incorrect masking"

  Scenario: control frames cannot be fragmented
    Given the wire header bytes "0980aabbccdd"
    When a server parses the header
    Then parsing fails with "fragmented control"

  Scenario: reserved bits are rejected without a negotiated extension
    Given the wire header bytes "a28001020304"
    When a server parses the header
    Then parsing fails with "reserved bits"

  Scenario: extended lengths must use their canonical encoding
    Given the wire header bytes "82fe007d01020304"
    When a server parses the header
    Then parsing fails with "non-canonical"

  Scenario: control payloads cannot exceed 125 bytes
    Given the wire header bytes "89fe007e01020304"
    When a server parses the header
    Then parsing fails with "control payload"

  Scenario: payload masking is involutive
    Given payload "The quick brown fox" and mask "37fa213d"
    When I apply the mask twice
    Then the payload is unchanged

  Scenario: control frames interleave a fragmented text message
    Given an open message assembler
    When I feed non-final text "hello "
    And I feed a ping "still here"
    And I feed final continuation "world"
    Then the completed text is "hello world"

  Scenario: invalid UTF-8 fails a text message
    Given an open message assembler
    When I feed final text bytes "c328"
    Then assembly fails with "invalid UTF-8"

  Scenario Outline: invalid close codes are rejected
    When I parse a close payload with code <code>
    Then close parsing fails

    Examples:
      | code |
      | 999  |
      | 1005 |
      | 1006 |
      | 2000 |
      | 5000 |
