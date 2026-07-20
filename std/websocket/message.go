package websocket

import (
	"encoding/binary"
	"errors"
	"unicode/utf8"
)

type CloseCode uint16

const (
	CloseNormalClosure   CloseCode = 1000
	CloseGoingAway       CloseCode = 1001
	CloseProtocolError   CloseCode = 1002
	CloseUnsupportedData CloseCode = 1003
	CloseInvalidPayload  CloseCode = 1007
	ClosePolicyViolation CloseCode = 1008
	CloseMessageTooBig   CloseCode = 1009
	CloseMandatoryExt    CloseCode = 1010
	CloseInternalError   CloseCode = 1011
)

var (
	ErrUnexpectedContinuation = errors.New("websocket: unexpected continuation")
	ErrExpectedContinuation   = errors.New("websocket: expected continuation")
	ErrInvalidUTF8            = errors.New("websocket: invalid UTF-8")
	ErrInvalidClosePayload    = errors.New("websocket: invalid close payload")
	ErrInvalidCloseCode       = errors.New("websocket: invalid close code")
	ErrMessageTooLarge        = errors.New("websocket: message exceeds configured limit")
)

func ValidCloseCode(code CloseCode) bool {
	if code >= 3000 && code <= 4999 {
		return true
	}
	if code < 1000 || code > 1014 {
		return false
	}
	switch code {
	case 1004, 1005, 1006:
		return false
	default:
		return true
	}
}

func ParseClosePayload(payload []byte) (CloseCode, string, error) {
	if len(payload) == 0 {
		return 0, "", nil
	}
	if len(payload) == 1 {
		return 0, "", ErrInvalidClosePayload
	}
	code := CloseCode(binary.BigEndian.Uint16(payload[:2]))
	if !ValidCloseCode(code) {
		return 0, "", &failureDetail{cause: ErrInvalidCloseCode, code: code}
	}
	reason := payload[2:]
	if !utf8.Valid(reason) {
		return 0, "", ErrInvalidUTF8
	}
	return code, string(reason), nil
}

func AppendClosePayload(dst []byte, code CloseCode, reason string) ([]byte, error) {
	if code == 0 {
		if reason != "" {
			return dst, ErrInvalidClosePayload
		}
		return dst, nil
	}
	if !ValidCloseCode(code) {
		return dst, ErrInvalidCloseCode
	}
	if !utf8.ValidString(reason) {
		return dst, ErrInvalidUTF8
	}
	if len(reason) > 123 {
		return dst, ErrControlTooLarge
	}
	dst = append(dst, byte(code>>8), byte(code))
	dst = append(dst, reason...)
	return dst, nil
}

// Assembler validates fragmentation and turns frames into complete messages.
// A nil message means a non-final data fragment was accepted.
type Assembler struct {
	MaxMessage int64
	op         Opcode
	fragmented bool
	compressed bool
	payload    []byte
	inflate    func([]byte, int64) ([]byte, error)
}

func (a *Assembler) Reset() {
	a.op, a.fragmented, a.compressed = 0, false, false
	a.payload = a.payload[:0]
}

func (a *Assembler) Feed(h Header, payload []byte) (Message, error) {
	return a.feed(h, payload, true)
}

// feed accepts ownership of payload when copyPayload is false. Conn uses that
// path because each frame already has a private read buffer; public Feed keeps
// its defensive-copy contract for low-level callers.
func (a *Assembler) feed(h Header, payload []byte, copyPayload bool) (Message, error) {
	if int64(len(payload)) != h.Length {
		return nil, ErrNeedMoreData
	}
	switch h.Opcode {
	case OpPing:
		if h.RSV1 {
			return nil, ErrReservedBits
		}
		return PingMessage{Payload: ownPayload(payload, copyPayload)}, nil
	case OpPong:
		if h.RSV1 {
			return nil, ErrReservedBits
		}
		return PongMessage{Payload: ownPayload(payload, copyPayload)}, nil
	case OpClose:
		if h.RSV1 {
			return nil, ErrReservedBits
		}
		code, reason, err := ParseClosePayload(payload)
		if err != nil {
			return nil, err
		}
		return CloseMessage{Code: code, Reason: reason}, nil
	case OpContinuation:
		if h.RSV1 {
			return nil, ErrReservedBits
		}
		if !a.fragmented {
			return nil, ErrUnexpectedContinuation
		}
		if err := a.append(payload); err != nil {
			return nil, err
		}
		if !h.FIN {
			return nil, nil
		}
		op := a.op
		body := a.payload
		a.payload = nil
		compressed := a.compressed
		a.Reset()
		if compressed {
			if a.inflate == nil {
				return nil, ErrReservedBits
			}
			var err error
			body, err = a.inflate(body, a.MaxMessage)
			if err != nil {
				return nil, err
			}
		}
		return messageFor(op, body)
	case OpText, OpBinary:
		if a.fragmented {
			return nil, ErrExpectedContinuation
		}
		if h.FIN {
			if a.MaxMessage > 0 && int64(len(payload)) > a.MaxMessage {
				return nil, &failureDetail{cause: ErrMessageTooLarge, limit: a.MaxMessage}
			}
			body := ownPayload(payload, copyPayload)
			if h.RSV1 {
				if a.inflate == nil {
					return nil, ErrReservedBits
				}
				var err error
				body, err = a.inflate(body, a.MaxMessage)
				if err != nil {
					return nil, err
				}
			}
			return messageFor(h.Opcode, body)
		}
		a.op, a.fragmented, a.compressed = h.Opcode, true, h.RSV1
		a.payload = a.payload[:0]
		if err := a.append(payload); err != nil {
			a.Reset()
			return nil, err
		}
		return nil, nil
	default:
		return nil, ErrInvalidOpcode
	}
}

func ownPayload(payload []byte, copyPayload bool) []byte {
	if !copyPayload {
		return payload
	}
	return append([]byte(nil), payload...)
}

func (a *Assembler) append(payload []byte) error {
	if a.MaxMessage > 0 && int64(len(a.payload))+int64(len(payload)) > a.MaxMessage {
		return &failureDetail{cause: ErrMessageTooLarge, limit: a.MaxMessage}
	}
	a.payload = append(a.payload, payload...)
	return nil
}

func messageFor(op Opcode, payload []byte) (Message, error) {
	if op == OpText {
		if !utf8.Valid(payload) {
			return nil, ErrInvalidUTF8
		}
		return TextMessage{Payload: payload}, nil
	}
	return BinaryMessage{Payload: payload}, nil
}
