package http3

import (
	"context"
	"crypto/tls"
	"io"

	xquic "goforge.dev/goplus/std/internal/quic"
)

type xnetConn struct {
	conn   *xquic.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func newXNetConn(conn *xquic.Conn) *xnetConn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &xnetConn{conn: conn, ctx: ctx, cancel: cancel}
	go func() {
		_ = conn.Wait(ctx)
		cancel()
	}()
	return c
}

func (c *xnetConn) Context() context.Context { return c.ctx }
func (c *xnetConn) openUni(ctx context.Context) (wireSendStream, error) {
	stream, err := c.conn.NewSendOnlyStream(ctx)
	if err != nil {
		return nil, err
	}
	stream.SetWriteContext(ctx)
	return xnetSendStream{stream}, nil
}
func (c *xnetConn) acceptUni(ctx context.Context) (io.Reader, error) {
	stream, err := c.conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	if !stream.IsReadOnly() {
		stream.Reset(0x103)
		return nil, &xquic.ApplicationError{Code: 0x103, Reason: "server created bidirectional stream"}
	}
	stream.SetReadContext(ctx)
	return stream, nil
}
func (c *xnetConn) openStream(ctx context.Context) (wireStream, error) {
	stream, err := c.conn.NewStream(ctx)
	if err != nil {
		return nil, err
	}
	stream.SetReadContext(ctx)
	stream.SetWriteContext(ctx)
	return &xnetStream{stream}, nil
}
func (c *xnetConn) closeWithError(code uint64, reason string) error {
	c.conn.Abort(&xquic.ApplicationError{Code: code, Reason: reason})
	c.cancel()
	return nil
}
func (c *xnetConn) tlsState() tls.ConnectionState { return c.conn.ConnectionState() }

type xnetSendStream struct{ stream *xquic.Stream }

func (s xnetSendStream) Write(p []byte) (int, error) { return s.stream.Write(p) }
func (s xnetSendStream) Close() error                { s.stream.CloseWrite(); return nil }
func (s xnetSendStream) flush() error                { return s.stream.Flush() }

type xnetStream struct{ *xquic.Stream }

func (s *xnetStream) Close() error            { s.CloseWrite(); return nil }
func (s *xnetStream) flush() error            { return s.Flush() }
func (s *xnetStream) needsContextWatch() bool { return false }
func (s *xnetStream) abortRead(uint64)        { s.CloseRead() }
func (s *xnetStream) abortWrite(code uint64)  { s.Reset(code) }
