package websocket

import "errors"

type failureDetail struct {
	cause        error
	opcode       byte
	bits         byte
	expectMasked bool
	code         CloseCode
	limit        int64
}

func (e *failureDetail) Error() string { return e.cause.Error() }
func (e *failureDetail) Unwrap() error { return e.cause }

// FailureOf lifts an ordinary Go error into the closed Go+ Failure vocabulary.
// It lets Go+ callers exhaustively fold protocol failures while preserving an
// unknown transport error for ordinary errors.Is/errors.As handling.
func FailureOf(err error) Failure {
	var detail *failureDetail
	_ = errors.As(err, &detail)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNeedMoreData):
		return NeedMoreData{}
	case errors.Is(err, ErrInvalidOpcode):
		if detail != nil {
			return InvalidOpcode{Opcode: detail.opcode}
		}
		return InvalidOpcode{}
	case errors.Is(err, ErrReservedBits):
		if detail != nil {
			return ReservedBits{Bits: detail.bits}
		}
		return ReservedBits{}
	case errors.Is(err, ErrWrongMask):
		if detail != nil {
			return WrongMask{ExpectMasked: detail.expectMasked}
		}
		return WrongMask{}
	case errors.Is(err, ErrNonCanonicalLength), errors.Is(err, ErrInvalidLength):
		return NonCanonicalLength{}
	case errors.Is(err, ErrControlFragmented):
		return ControlFragmented{}
	case errors.Is(err, ErrControlTooLarge):
		return ControlTooLarge{}
	case errors.Is(err, ErrUnexpectedContinuation):
		return UnexpectedContinuation{}
	case errors.Is(err, ErrExpectedContinuation):
		return ExpectedContinuation{}
	case errors.Is(err, ErrInvalidUTF8):
		return InvalidUTF8{}
	case errors.Is(err, ErrInvalidClosePayload):
		return InvalidClosePayload{}
	case errors.Is(err, ErrInvalidCloseCode):
		if detail != nil {
			return InvalidCloseCode{Code: detail.code}
		}
		return InvalidCloseCode{}
	case errors.Is(err, ErrMessageTooLarge):
		if detail != nil {
			return MessageTooLarge{Limit: detail.limit}
		}
		return MessageTooLarge{}
	case errors.Is(err, ErrHandshake), errors.Is(err, ErrInvalidExtension):
		return HandshakeRejected{Reason: err.Error()}
	default:
		return TransportFailed{Err: err}
	}
}
