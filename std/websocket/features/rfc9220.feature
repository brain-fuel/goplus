Feature: RFC 9220 WebSockets over HTTP/3
  The ordinary Dial and Upgrade APIs use HTTP/3 extended CONNECT when origin
  capability is known, while retaining HTTP/2 and HTTP/1.1 compatibility.

  Scenario: HTTP 3 extended CONNECT replaces HTTP 1.1 upgrade fields
    Given a canonical RFC 9220 opening request
    When I validate the HTTP 3 extended CONNECT request
    Then extended CONNECT validation succeeds
    And the request has no RFC 6455 connection fields

  Scenario Outline: forbidden HTTP 1.1 fields are rejected over HTTP 3
    Given a canonical RFC 9220 opening request
    And the extended CONNECT request has forbidden header <header>
    When I validate the HTTP 3 extended CONNECT request
    Then opening validation fails

    Examples:
      | header               |
      | Connection           |
      | Upgrade              |
      | Sec-WebSocket-Key    |
      | Sec-WebSocket-Accept |
      | :protocol            |

  Scenario: HTTP 3 uses the websocket protocol pseudo-header
    Given a canonical RFC 9220 opening request
    And the HTTP 3 protocol pseudo-header is not websocket
    When I validate the HTTP 3 extended CONNECT request
    Then opening validation fails

  Scenario: learned secure capability prefers RFC 9220
    Given a secure WebSocket URL with learned HTTP 3 capability
    Then RFC 9220 is attempted before RFC 8441 and RFC 6455
