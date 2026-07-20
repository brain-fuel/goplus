package http3

import (
	"context"
	"crypto/tls"
	"io"

	"github.com/quic-go/quic-go"
)

type wireStream interface {
	io.ReadWriteCloser
	flush() error
	needsContextWatch() bool
	abortRead(uint64)
	abortWrite(uint64)
}

type wireSendStream interface {
	io.WriteCloser
	flush() error
}

type wireConn interface {
	Context() context.Context
	openUni(context.Context) (wireSendStream, error)
	acceptUni(context.Context) (io.Reader, error)
	openStream(context.Context) (wireStream, error)
	closeWithError(uint64, string) error
	tlsState() tls.ConnectionState
}

type quicGoConn struct{ conn *quic.Conn }

func (c *quicGoConn) Context() context.Context { return c.conn.Context() }
func (c *quicGoConn) openUni(ctx context.Context) (wireSendStream, error) {
	stream, err := c.conn.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return quicGoSendStream{SendStream: stream}, nil
}
func (c *quicGoConn) acceptUni(ctx context.Context) (io.Reader, error) {
	return c.conn.AcceptUniStream(ctx)
}
func (c *quicGoConn) openStream(ctx context.Context) (wireStream, error) {
	stream, err := c.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &quicGoStream{Stream: stream}, nil
}
func (c *quicGoConn) closeWithError(code uint64, reason string) error {
	return c.conn.CloseWithError(quic.ApplicationErrorCode(code), reason)
}
func (c *quicGoConn) tlsState() tls.ConnectionState { return c.conn.ConnectionState().TLS }

type quicGoStream struct{ *quic.Stream }

type quicGoSendStream struct{ *quic.SendStream }

func (s quicGoSendStream) flush() error { return nil }

func (s *quicGoStream) abortRead(code uint64)   { s.CancelRead(quic.StreamErrorCode(code)) }
func (s *quicGoStream) abortWrite(code uint64)  { s.CancelWrite(quic.StreamErrorCode(code)) }
func (s *quicGoStream) flush() error            { return nil }
func (s *quicGoStream) needsContextWatch() bool { return true }
