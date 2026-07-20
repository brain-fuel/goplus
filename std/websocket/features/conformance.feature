Feature: Autobahn conformance discipline
  Every Autobahn category is represented by an implementation invariant and
  the checked-in runner fails unless every required case is successful.

  Scenario Outline: required Autobahn categories
    Given Autobahn category <category>
    Then the conformance manifest marks it required

    Examples:
      | category                    |
      | framing                     |
      | ping-pong                   |
      | reserved-bits               |
      | opcodes                     |
      | fragmentation               |
      | utf8                        |
      | limits                      |
      | closing-handshake           |
      | opening-handshake           |
      | permessage-deflate          |
