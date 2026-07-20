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

// Session is a proof token used by Go+ protocol orchestration. It erases to a
// compact Go interface; network ownership remains in Conn.
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
