package http3

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"

	quic "github.com/quic-go/quic-go"
)

// QUICGoServer runs Go+'s native RFC 9114 server over quic-go's RFC 9000
// transport. This is useful when applications want quic-go's mature packet
// engine without using its HTTP/3 implementation.
type QUICGoServer struct {
	Handler    http.Handler
	TLSConfig  *tls.Config
	QUICConfig *quic.Config

	mu        sync.Mutex
	listener  *quic.Listener
	transport *quic.Transport
	closed    bool
}

// Serve serves HTTP/3 on packetConn and owns the packet connection.
func (s *QUICGoServer) Serve(packetConn net.PacketConn) error {
	if packetConn == nil {
		return errors.New("http3: nil packet connection")
	}
	if s.TLSConfig == nil {
		return errors.New("http3: TLSConfig is required")
	}
	tlsConfig := s.TLSConfig.Clone()
	if tlsConfig.MinVersion < tls.VersionTLS13 {
		tlsConfig.MinVersion = tls.VersionTLS13
	}
	tlsConfig.NextProtos = []string{"h3"}
	config := s.QUICConfig
	if config == nil {
		config = new(quic.Config)
	} else {
		config = config.Clone()
	}
	transport := &quic.Transport{Conn: packetConn}
	listener, err := transport.Listen(tlsConfig, config)
	if err != nil {
		_ = transport.Close()
		return err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = listener.Close()
		_ = transport.Close()
		return http.ErrServerClosed
	}
	s.listener, s.transport = listener, transport
	s.mu.Unlock()
	handler := s.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return http.ErrServerClosed
			}
			return err
		}
		go serveQUICGoConn(conn, handler)
	}
}

func (s *QUICGoServer) Close() error {
	s.mu.Lock()
	s.closed = true
	listener, transport := s.listener, s.transport
	s.mu.Unlock()
	if listener != nil {
		_ = listener.Close()
	}
	if transport != nil {
		return transport.Close()
	}
	return nil
}

func (s *QUICGoServer) Shutdown(context.Context) error { return s.Close() }

type quicGoServerConn struct {
	conn *quic.Conn
}

func newQUICGoServerConn(conn *quic.Conn) *quicGoServerConn {
	return &quicGoServerConn{conn: conn}
}

func serveQUICGoConn(conn *quic.Conn, handler http.Handler) {
	c := newQUICGoServerConn(conn)
	ctx := conn.Context()
	if err := writeServerCriticalStreams(ctx, c); err != nil {
		c.abort(0x102, err.Error())
		return
	}
	uniState := newServerUniState()
	go func() {
		for {
			stream, err := conn.AcceptUniStream(ctx)
			if err != nil {
				return
			}
			go func() {
				if code, err := consumeClientUniStream(stream, uniState); err != nil {
					c.abort(code, err.Error())
				}
			}()
		}
	}()
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}
		go serveRequestStream(c, quicGoServerStream{stream}, handler, ctx, uniState)
	}
}

func (c *quicGoServerConn) Context() context.Context { return c.conn.Context() }
func (c *quicGoServerConn) accept(ctx context.Context) (acceptedServerStream, error) {
	stream, err := c.conn.AcceptStream(ctx)
	if err != nil {
		return acceptedServerStream{}, err
	}
	return acceptedServerStream{bidi: quicGoServerStream{stream}}, nil
}
func (c *quicGoServerConn) newSendOnly(ctx context.Context) (serverSendStream, error) {
	stream, err := c.conn.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return quicGoServerSendStream{stream}, nil
}
func (c *quicGoServerConn) abort(code uint64, reason string) {
	_ = c.conn.CloseWithError(quic.ApplicationErrorCode(code), reason)
}
func (c *quicGoServerConn) remoteAddr() string            { return c.conn.RemoteAddr().String() }
func (c *quicGoServerConn) tlsState() tls.ConnectionState { return c.conn.ConnectionState().TLS }

type quicGoServerStream struct{ *quic.Stream }

func (s quicGoServerStream) flush() error { return nil }
func (s quicGoServerStream) closeRead() {
	s.CancelRead(quic.StreamErrorCode(errorRequestCancelled))
}
func (s quicGoServerStream) closeWrite() error { return s.Close() }
func (s quicGoServerStream) reset(code uint64) {
	s.CancelRead(quic.StreamErrorCode(code))
	s.CancelWrite(quic.StreamErrorCode(code))
}
func (s quicGoServerStream) setReadContext(ctx context.Context) {
	deadline, _ := ctx.Deadline()
	_ = s.SetReadDeadline(deadline)
}
func (s quicGoServerStream) setWriteContext(ctx context.Context) {
	deadline, _ := ctx.Deadline()
	_ = s.SetWriteDeadline(deadline)
}

type quicGoServerSendStream struct{ *quic.SendStream }

func (s quicGoServerSendStream) flush() error { return nil }

var _ serverConn = (*quicGoServerConn)(nil)
var _ serverStream = quicGoServerStream{}
var _ serverSendStream = quicGoServerSendStream{}
