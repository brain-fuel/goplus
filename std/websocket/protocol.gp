// Package websocket implements RFC 6455 WebSocket framing and transport.
// The semantic layer is authored in Go+; hot framing paths remain small,
// allocation-free Go functions so both Go+ and Go callers get the same wire.
package websocket

// Message is the complete application/control message vocabulary.
//goplus:derive off
type Message enum {
	TextMessage(payload []byte)
	BinaryMessage(payload []byte)
	PingMessage(payload []byte)
	PongMessage(payload []byte)
	CloseMessage(code CloseCode, reason string)
}

// Failure classifies protocol failure without string matching.
type Failure enum {
	NeedMoreData
	InvalidOpcode(opcode byte)
	ReservedBits(bits byte)
	WrongMask(expectMasked bool)
	NonCanonicalLength
	ControlFragmented
	ControlTooLarge
	UnexpectedContinuation
	ExpectedContinuation
	InvalidUTF8
	InvalidClosePayload
	InvalidCloseCode(code CloseCode)
	MessageTooLarge(limit int64)
	HandshakeRejected(status int, reason string)
	TransportFailed(err error)
}

// Phase indexes the protocol transitions that are valid for a connection.
type Phase enum { ConnectingPhase; OpenPhase; CloseSentPhase; CloseReceivedPhase; ClosedPhase }

// Session is the lightweight proof-token API retained for v0.18
// compatibility. New Go+ orchestration should prefer Capability, which also
// carries linear connection ownership.
//goplus:derive off
type Session[p Phase] enum {
	ConnectingSession() Session[ConnectingPhase]
	OpenSession() Session[OpenPhase]
	CloseSentSession() Session[CloseSentPhase]
	CloseReceivedSession() Session[CloseReceivedPhase]
	ClosedSession() Session[ClosedPhase]
}

func Open(_ Session[ConnectingPhase]) Session[OpenPhase] { return OpenSession() }
func SentClose(_ Session[OpenPhase]) Session[CloseSentPhase] { return CloseSentSession() }
func ReceivedClose(_ Session[OpenPhase]) Session[CloseReceivedPhase] { return CloseReceivedSession() }
func FinishSent(_ Session[CloseSentPhase]) Session[ClosedPhase] { return ClosedSession() }
func FinishReceived(_ Session[CloseReceivedPhase]) Session[ClosedPhase] { return ClosedSession() }

// Capability is the Go+ ownership-oriented API. Go callers that do not want
// typestate use Conn directly.
type Capability[p Phase] enum {
	OpenCapability(conn *Conn) Capability[OpenPhase]
	CloseSentCapability(conn *Conn) Capability[CloseSentPhase]
	ClosedCapability(conn *Conn) Capability[ClosedPhase]
}

// CloseAttempt preserves ownership on both the success and failure paths.
//goplus:derive off
type CloseAttempt enum {
	CloseStarted(capability Capability[CloseSentPhase])
	CloseFailed(capability Capability[OpenPhase], err error)
}

// Send consumes and returns the open capability, preventing concurrent or
// accidental duplicated ownership in Go+ orchestration.
func Send(1 capability Capability[OpenPhase], message Message) (Capability[OpenPhase], error) {
	conn := capabilityConn(capability)
	err := conn.WriteMessage(message)
	return OpenCapability(conn), err
}

func BeginClose(1 capability Capability[OpenPhase], code CloseCode, reason string) CloseAttempt {
	conn := capabilityConn(capability)
	err := conn.WriteClose(code, reason)
	if err != nil {
		return CloseFailed(OpenCapability(conn), err)
	}
	return CloseStarted(CloseSentCapability(conn))
}

func FinishClose(1 capability Capability[CloseSentPhase]) (Capability[ClosedPhase], error) {
	conn := capabilityConn(capability)
	err := conn.Close()
	return ClosedCapability(conn), err
}
