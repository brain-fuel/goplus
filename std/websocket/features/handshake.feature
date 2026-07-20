Feature: RFC 6455 opening handshake
  Opening handshakes reject ambiguity before a connection changes protocol,
  while token-valued HTTP fields may be safely split across repeated lines.

  Scenario: a canonical request is accepted
    Given a canonical WebSocket opening request
    When I validate the opening request
    Then opening validation succeeds

  Scenario: repeated token fields are joined
    Given a canonical WebSocket opening request
    And Connection is split across repeated header fields
    When I validate the opening request
    Then opening validation succeeds

  Scenario Outline: security-sensitive singleton fields reject ambiguity
    Given a canonical WebSocket opening request
    And the request has duplicate <header> fields
    When I validate the opening request
    Then opening validation fails

    Examples:
      | header                |
      | Sec-WebSocket-Key     |
      | Sec-WebSocket-Version |

  Scenario: only GET may upgrade
    Given a canonical WebSocket opening request
    And the request method is POST
    When I validate the opening request
    Then opening validation fails

  Scenario: HTTP 1.1 Host is mandatory
    Given a canonical WebSocket opening request
    And the request Host is empty
    When I validate the opening request
    Then opening validation fails
