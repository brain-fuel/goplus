Feature: Enum-tag indices (typestate)
  A type-parameter-less enum whose variants are all bare is a
  first-order INDEX DOMAIN: its tags may index other enums
  (`Socket[Open]`), giving typestate-shaped declarations. Tag indices
  erase exactly like nat indices; sorts are checked — a tag cannot sit
  in a nat position and vice versa.

  Background:
    Given a file "go.mod":
      """
      module example.com/demo

      go 1.24
      """

  Scenario: A typestate enum declares, erases, and crosses packages
    Given a G++ file "net/net.gpp":
      """
      package net

      type State enum {
      	Open
      	Closed
      }

      type Socket[s State] enum {
      	Fresh() Socket[Open]
      	Done(Reason string) Socket[Closed]
      }
      """
    And a G++ file "main.gpp":
      """
      package main

      import (
      	"fmt"

      	"example.com/demo/net"
      )

      func describe(s net.Socket[net.Open]) string {
      	match s {
      	case net.Fresh():
      		return "fresh"
      	case net.Done(r):
      		return r
      	}
      	return "?"
      }

      func main() {
      	fmt.Println(describe(net.Fresh()))
      }
      """
    When I run gpp with arguments "gen ./..."
    Then the exit code is 0
    When I run gpp with arguments "run ."
    Then the exit code is 0
    And stdout contains "fresh"
    And the file "net/net_gpp.go" contains:
      """
      //gpp:enum Socket[s State]
      type Socket interface{ isSocket() }
      """
    And the file "main_gpp.go" contains:
      """
      func describe(s net.Socket) string {
      """

  Scenario: A non-tag in a tag-indexed position is rejected
    Given a G++ file "main.gpp":
      """
      package main

      type State enum {
      	Open
      	Closed
      }

      type Socket[s State] enum {
      	Bad() Socket[5]
      }

      func main() {}
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "index argument 5 is not a State tag or a State-sorted index parameter"

  Scenario: A tag in a nat-indexed position is rejected
    Given a G++ file "main.gpp":
      """
      package main

      type State enum {
      	Open
      	Closed
      }

      type V[n nat] enum {
      	Bad() V[Open]
      }

      func main() {}
      """
    When I run gpp with arguments "gen ."
    Then the exit code is 2
    And stderr contains "index argument Open uses tag Open in a nat-indexed position"
