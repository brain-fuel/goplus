package http3

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	xquic "goforge.dev/goplus/std/internal/quic"
)

// NativeServer is an RFC 9114 server running on the Go+ owned RFC 9000 engine.
// It is independent of quic-go's server so the Go+ HTTP/3 data path can be
// deployed and measured without the reference implementation.
type NativeServer struct {
	Handler    http.Handler
	TLSConfig  *tls.Config
	QUICConfig *xquic.Config

	mu       sync.Mutex
	endpoint *xquic.Endpoint
	closed   bool
}

// Serve serves HTTP/3 on packetConn and owns the packet connection.
func (s *NativeServer) Serve(packetConn net.PacketConn) error {
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
		config = new(xquic.Config)
	} else {
		config = config.Clone()
	}
	config.TLSConfig = tlsConfig
	endpoint, err := xquic.NewEndpoint(packetConn, config)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = endpoint.Close(ctx)
		return http.ErrServerClosed
	}
	s.endpoint = endpoint
	s.mu.Unlock()
	handler := s.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	for {
		conn, err := endpoint.Accept(context.Background())
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return http.ErrServerClosed
			}
			return err
		}
		go serveXNetConn(conn, handler)
	}
}

// Close aborts active connections and stops accepting new ones.
func (s *NativeServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}

// Shutdown stops accepting connections and waits for active QUIC connections
// to close until ctx expires.
func (s *NativeServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	endpoint := s.endpoint
	s.mu.Unlock()
	if endpoint == nil {
		return nil
	}
	err := endpoint.Close(ctx)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

// XNetServer is retained as a compatibility spelling for NativeServer.
// Deprecated: use NativeServer.
type XNetServer = NativeServer

type serverStream interface {
	io.Reader
	io.Writer
	flush() error
	closeRead()
	closeWrite() error
	reset(uint64)
	setReadContext(context.Context)
	setWriteContext(context.Context)
}

type serverSendStream interface {
	io.Writer
	flush() error
}

type acceptedServerStream struct {
	uni  io.Reader
	bidi serverStream
}

type serverConn interface {
	Context() context.Context
	accept(context.Context) (acceptedServerStream, error)
	newSendOnly(context.Context) (serverSendStream, error)
	abort(uint64, string)
	remoteAddr() string
	tlsState() tls.ConnectionState
}

type xnetServerConn struct {
	conn *xquic.Conn
	ctx  context.Context
}

func newXNetServerConn(conn *xquic.Conn) xnetServerConn {
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = conn.Wait(ctx); cancel() }()
	return xnetServerConn{conn: conn, ctx: ctx}
}
func (c xnetServerConn) Context() context.Context { return c.ctx }
func (c xnetServerConn) accept(ctx context.Context) (acceptedServerStream, error) {
	stream, err := c.conn.AcceptStream(ctx)
	if err != nil {
		return acceptedServerStream{}, err
	}
	if stream.IsReadOnly() {
		return acceptedServerStream{uni: stream}, nil
	}
	if stream.IsWriteOnly() {
		stream.Reset(0x103)
		return acceptedServerStream{}, errors.New("http3: client opened write-only stream")
	}
	return acceptedServerStream{bidi: xnetServerStream{stream}}, nil
}
func (c xnetServerConn) newSendOnly(ctx context.Context) (serverSendStream, error) {
	stream, err := c.conn.NewSendOnlyStream(ctx)
	if err != nil {
		return nil, err
	}
	stream.SetWriteContext(ctx)
	return xnetServerSendStream{stream}, nil
}
func (c xnetServerConn) abort(code uint64, reason string) {
	c.conn.Abort(&xquic.ApplicationError{Code: code, Reason: reason})
}
func (c xnetServerConn) remoteAddr() string            { return c.conn.RemoteAddr().String() }
func (c xnetServerConn) tlsState() tls.ConnectionState { return c.conn.ConnectionState() }

type xnetServerStream struct{ *xquic.Stream }

func (s xnetServerStream) flush() error                        { return s.Flush() }
func (s xnetServerStream) closeRead()                          { s.CloseRead() }
func (s xnetServerStream) closeWrite() error                   { s.CloseWrite(); return nil }
func (s xnetServerStream) reset(code uint64)                   { s.Reset(code) }
func (s xnetServerStream) setReadContext(ctx context.Context)  { s.SetReadContext(ctx) }
func (s xnetServerStream) setWriteContext(ctx context.Context) { s.SetWriteContext(ctx) }

type xnetServerSendStream struct{ *xquic.Stream }

func (s xnetServerSendStream) flush() error { return s.Flush() }

func serveXNetConn(conn *xquic.Conn, handler http.Handler) {
	serveServerConn(newXNetServerConn(conn), handler)
}

func serveServerConn(conn serverConn, handler http.Handler) {
	ctx := conn.Context()
	if err := writeServerCriticalStreams(ctx, conn); err != nil {
		conn.abort(0x102, err.Error())
		return
	}
	uniState := newServerUniState()
	acceptAndServeRequests(conn, handler, ctx, uniState)
}

func acceptAndServeRequests(conn serverConn, handler http.Handler, ctx context.Context, uniState *serverUniState) {
	for {
		stream, err := conn.accept(ctx)
		if err != nil {
			return
		}
		if stream.uni != nil {
			go func() {
				if code, err := consumeClientUniStream(stream.uni, uniState); err != nil {
					conn.abort(code, err.Error())
				}
			}()
			continue
		}
		// Preserve multiplexing by installing the next accept waiter before
		// serving this stream, while keeping the current stream on the hot
		// goroutine that was just awakened by QUIC.
		go acceptAndServeRequests(conn, handler, ctx, uniState)
		serveRequestStream(conn, stream.bidi, handler, ctx, uniState)
		return
	}
}

type serverUniState struct {
	mu                        sync.Mutex
	control, encoder, decoder bool
	peerMaxField              atomic.Uint64
}

func newServerUniState() *serverUniState {
	state := new(serverUniState)
	state.peerMaxField.Store(^uint64(0))
	return state
}

func (s *serverUniState) claim(streamType uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var seen *bool
	switch streamType {
	case streamTypeControl:
		seen = &s.control
	case streamTypeQPACKEncoder:
		seen = &s.encoder
	case streamTypeQPACKDecoder:
		seen = &s.decoder
	default:
		return nil
	}
	if *seen {
		return fmt.Errorf("http3: duplicate client critical stream type %d", streamType)
	}
	*seen = true
	return nil
}

func writeServerCriticalStreams(ctx context.Context, conn serverConn) error {
	control, err := conn.newSendOnly(ctx)
	if err != nil {
		return err
	}
	payload := appendQUICVarint(nil, settingExtendedConnect)
	payload = appendQUICVarint(payload, 1)
	payload = appendQUICVarint(payload, settingMaxFieldSection)
	payload = appendQUICVarint(payload, maxFieldSectionSize)
	prefix := appendQUICVarint(nil, streamTypeControl)
	prefix, _ = AppendFrameHeader(prefix, frameTypeSettings, uint64(len(payload)))
	prefix = append(prefix, payload...)
	if _, err := control.Write(prefix); err != nil {
		return err
	}
	if err := control.flush(); err != nil {
		return err
	}
	for _, streamType := range []uint64{streamTypeQPACKEncoder, streamTypeQPACKDecoder} {
		stream, err := conn.newSendOnly(ctx)
		if err != nil {
			return err
		}
		if _, err := stream.Write(appendQUICVarint(nil, streamType)); err != nil {
			return err
		}
		if err := stream.flush(); err != nil {
			return err
		}
	}
	return nil
}

func consumeClientUniStream(stream io.Reader, state *serverUniState) (uint64, error) {
	reader := &byteReader{Reader: stream}
	streamType, err := readQUICVarint(reader)
	if err != nil {
		return errorClosedCritical, err
	}
	if err := state.claim(streamType); err != nil {
		return 0x103, err
	}
	if streamType == streamTypePush {
		return 0x103, errors.New("http3: client created push stream")
	}
	if streamType == streamTypeControl {
		frameType, err := readQUICVarint(reader)
		if err != nil {
			return errorClosedCritical, err
		}
		length, err := readQUICVarint(reader)
		if err != nil || frameType != frameTypeSettings {
			return errorFrameUnexpected, errors.New("http3: client control stream did not begin with SETTINGS")
		}
		if length > 8<<10 {
			return errorSettings, errors.New("http3: client SETTINGS frame too large")
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return errorClosedCritical, err
		}
		settings, err := parseSettings(payload)
		if err != nil {
			return errorSettings, err
		}
		state.peerMaxField.Store(settings.maxFieldSection)
		for {
			frameType, err := readQUICVarint(reader)
			if err != nil {
				return errorClosedCritical, errors.New("http3: client control stream closed")
			}
			length, err := readQUICVarint(reader)
			if err != nil {
				return errorClosedCritical, err
			}
			switch frameType {
			case frameTypeSettings:
				return errorSettings, errors.New("http3: duplicate client SETTINGS")
			case frameTypeData, frameTypeHeaders, 0x5:
				return errorFrameUnexpected, errors.New("http3: forbidden frame on client control stream")
			}
			if _, err := io.CopyN(io.Discard, reader, int64(length)); err != nil {
				return errorClosedCritical, err
			}
		}
	}
	if streamType == streamTypeQPACKEncoder || streamType == streamTypeQPACKDecoder {
		_, _ = io.Copy(io.Discard, reader)
		return errorClosedCritical, errors.New("http3: client QPACK stream closed")
	}
	_, err = io.Copy(io.Discard, reader)
	return 0, err
}

func serveRequestStream(conn serverConn, stream serverStream, handler http.Handler, connCtx context.Context, uniState *serverUniState) {
	ctx, cancel := context.WithCancel(connCtx)
	defer cancel()
	stream.setReadContext(ctx)
	stream.setWriteContext(ctx)
	request, reader, err := readServerRequest(conn, stream, ctx)
	if err != nil {
		stream.reset(errorMessage)
		return
	}
	request.Body = &serverRequestBody{
		stream: stream, reader: reader, request: request,
		expected: request.ContentLength, enforceLength: request.ContentLength >= 0,
	}
	defer func() {
		_, _ = io.Copy(io.Discard, request.Body)
		_ = request.Body.Close()
	}()
	w := &xResponseWriter{stream: stream, requestMethod: request.Method, ctx: ctx, cancel: cancel, peerMaxField: &uniState.peerMaxField}
	defer func() {
		if recover() != nil {
			stream.reset(0x102)
			return
		}
		if !w.wroteHeader {
			w.WriteHeader(http.StatusOK)
		}
		w.writeTrailers()
		_ = stream.closeWrite()
	}()
	handler.ServeHTTP(w, request)
}

func readServerRequest(conn serverConn, stream serverStream, ctx context.Context) (*http.Request, *byteReader, error) {
	reader := &byteReader{Reader: stream}
	frameType, err := readQUICVarint(reader)
	if err != nil {
		return nil, nil, err
	}
	length, err := readQUICVarint(reader)
	if err != nil || frameType != frameTypeHeaders {
		return nil, nil, errors.New("http3: request did not begin with HEADERS")
	}
	fields, err := readFieldSection(reader, length)
	if err != nil {
		return nil, nil, err
	}
	request := &http.Request{Proto: "HTTP/3.0", ProtoMajor: 3, ProtoMinor: 0}
	var scheme, authority, path, protocol string
	regular := false
	seen := make(map[string]bool)
	for _, field := range fields {
		if strings.HasPrefix(field.Name, ":") {
			if regular || seen[field.Name] {
				return nil, nil, errors.New("http3: invalid request pseudo-header")
			}
			seen[field.Name] = true
			switch field.Name {
			case ":method":
				request.Method = field.Value
			case ":scheme":
				scheme = field.Value
			case ":authority":
				authority = field.Value
			case ":path":
				path = field.Value
			case ":protocol":
				protocol = field.Value
			default:
				return nil, nil, errors.New("http3: unknown request pseudo-header")
			}
			continue
		}
		regular = true
		if field.Name != strings.ToLower(field.Name) || !validHeaderName(field.Name) || connectionSpecificField(field.Name) {
			return nil, nil, errors.New("http3: invalid request field")
		}
		if request.Header == nil {
			request.Header = make(http.Header)
		}
		request.Header.Add(textproto.CanonicalMIMEHeaderKey(field.Name), field.Value)
	}
	if request.Method == "" || authority == "" {
		return nil, nil, errors.New("http3: missing required request pseudo-header")
	}
	if request.Method != http.MethodConnect || protocol != "" {
		if scheme == "" || path == "" {
			return nil, nil, errors.New("http3: missing request scheme or path")
		}
	} else if scheme != "" || path != "" {
		return nil, nil, errors.New("http3: CONNECT included scheme or path")
	}
	if protocol != "" && request.Method != http.MethodConnect {
		return nil, nil, errors.New("http3: protocol on non-CONNECT request")
	}
	if path == "" {
		path = "/"
	}
	requestURL, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, nil, err
	}
	requestURL.Scheme, requestURL.Host = scheme, authority
	request.URL = requestURL
	request.Host = authority
	if protocol != "" {
		request.Proto = protocol
	}
	request.RemoteAddr = conn.remoteAddr()
	tlsState := conn.tlsState()
	request.TLS = &tlsState
	request = request.WithContext(ctx)
	request.ContentLength = -1
	if raw := request.Header.Get("Content-Length"); raw != "" && request.Method != http.MethodConnect {
		request.ContentLength, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || request.ContentLength < 0 {
			return nil, nil, errors.New("http3: invalid request content length")
		}
	}
	return request, reader, nil
}

type serverRequestBody struct {
	stream        serverStream
	reader        *byteReader
	request       *http.Request
	remaining     uint64
	trailers      bool
	expected      int64
	read          int64
	enforceLength bool
}

func (b *serverRequestBody) Read(p []byte) (int, error) {
	for b.remaining == 0 {
		frameType, err := readQUICVarint(b.reader)
		if err != nil {
			if err == io.EOF && b.enforceLength && b.read < b.expected {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		length, err := readQUICVarint(b.reader)
		if err != nil {
			return 0, err
		}
		switch frameType {
		case frameTypeData:
			if b.trailers {
				return 0, errors.New("http3: DATA after request trailers")
			}
			b.remaining = length
			if b.enforceLength && length > uint64(b.expected-b.read) {
				return 0, errors.New("http3: request body exceeds content length")
			}
			if length == 0 {
				continue
			}
		case frameTypeHeaders:
			if b.trailers {
				return 0, errors.New("http3: multiple request trailer sections")
			}
			fields, err := readFieldSection(b.reader, length)
			if err != nil {
				return 0, err
			}
			if b.request.Trailer == nil {
				b.request.Trailer = make(http.Header)
			}
			for _, field := range fields {
				if strings.HasPrefix(field.Name, ":") || !validHeaderName(field.Name) {
					return 0, errors.New("http3: invalid request trailer")
				}
				b.request.Trailer.Add(textproto.CanonicalMIMEHeaderKey(field.Name), field.Value)
			}
			b.trailers = true
			continue
		default:
			if _, err := io.CopyN(io.Discard, b.reader, int64(length)); err != nil {
				return 0, err
			}
			continue
		}
	}
	if uint64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.reader.Read(p)
	b.remaining -= uint64(n)
	b.read += int64(n)
	if err == io.EOF && b.enforceLength && b.read < b.expected {
		err = io.ErrUnexpectedEOF
	}
	return n, err
}

func (b *serverRequestBody) Close() error { b.stream.closeRead(); return nil }

type xResponseWriter struct {
	stream        serverStream
	header        http.Header
	requestMethod string
	ctx           context.Context
	cancel        context.CancelFunc
	readCancel    context.CancelFunc
	writeCancel   context.CancelFunc
	wroteHeader   bool
	status        int
	err           error
	trailerNames  map[string]struct{}
	peerMaxField  *atomic.Uint64
}

func (w *xResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *xResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	if status < 100 || status > 999 {
		panic("invalid HTTP status code")
	}
	if status == http.StatusSwitchingProtocols {
		w.err = errors.New("http3: status 101 is invalid")
		return
	}
	informational := status < 200
	if !informational {
		w.status = status
		w.trailerNames = declaredTrailers(w.header)
	}
	frame, err := encodeResponseHeadersFrame(status, w.requestMethod, w.header, w.trailerNames, w.peerMaxField.Load())
	if err == nil {
		_, err = w.stream.Write(frame)
	}
	w.err = err
	if informational && err == nil {
		w.err = w.stream.flush()
		return
	}
	w.wroteHeader = true
}

func encodeResponseHeadersFrame(status int, requestMethod string, header http.Header, trailerNames map[string]struct{}, peerLimit uint64) ([]byte, error) {
	frame := make([]byte, 9, 9+128)
	frame = append(frame, 0, 0)
	limit := min(peerLimit, uint64(maxFieldSectionSize))
	var decodedSize uint64
	appendOne := func(field *HeaderField) error {
		size := uint64(len(field.Name) + len(field.Value) + 32)
		if size > limit-decodedSize {
			return ErrFieldSectionTooLarge
		}
		decodedSize += size
		var err error
		frame, err = appendField(frame, field)
		return err
	}
	var err error
	statusField := HeaderField{Name: ":status", Value: strconv.Itoa(status)}
	if err = appendOne(&statusField); err != nil {
		return nil, err
	}
	for name, values := range header {
		lower := strings.ToLower(name)
		if connectionSpecificField(lower) || strings.HasPrefix(name, http.TrailerPrefix) {
			continue
		}
		if requestMethod == http.MethodConnect && status >= 200 && status < 300 && lower == "content-length" {
			continue
		}
		if _, trailer := trailerNames[textproto.CanonicalMIMEHeaderKey(name)]; trailer {
			continue
		}
		for _, value := range values {
			field := HeaderField{Name: lower, Value: value}
			if err = appendOne(&field); err != nil {
				return nil, err
			}
		}
	}
	return finishHeadersFrame(frame), nil
}

func declaredTrailers(header http.Header) map[string]struct{} {
	var trailers map[string]struct{}
	for _, line := range header.Values("Trailer") {
		for _, name := range strings.Split(line, ",") {
			name = textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(name))
			if name == "" || connectionSpecificField(strings.ToLower(name)) {
				continue
			}
			if trailers == nil {
				trailers = make(map[string]struct{})
			}
			trailers[name] = struct{}{}
		}
	}
	for name := range header {
		if strings.HasPrefix(name, http.TrailerPrefix) {
			canonical := textproto.CanonicalMIMEHeaderKey(strings.TrimPrefix(name, http.TrailerPrefix))
			if canonical != "" {
				if trailers == nil {
					trailers = make(map[string]struct{})
				}
				trailers[canonical] = struct{}{}
			}
		}
	}
	return trailers
}

func (w *xResponseWriter) writeTrailers() {
	if w.err != nil || len(w.trailerNames) == 0 {
		return
	}
	fields := make([]HeaderField, 0, len(w.trailerNames))
	for name := range w.trailerNames {
		values := w.header.Values(name)
		if prefixed := w.header.Values(http.TrailerPrefix + name); len(prefixed) != 0 {
			values = prefixed
		}
		for _, value := range values {
			fields = append(fields, HeaderField{Name: strings.ToLower(name), Value: value})
		}
	}
	if len(fields) == 0 {
		return
	}
	frame, err := encodeHeadersFrameLimit(fields, w.peerMaxField.Load())
	if err == nil {
		_, err = w.stream.Write(frame)
	}
	w.err = err
}

func (w *xResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.err != nil {
		return 0, w.err
	}
	if w.requestMethod == http.MethodHead || w.status == http.StatusNoContent || w.status == http.StatusNotModified {
		return len(p), nil
	}
	var prefix [9]byte
	n := EncodeDataFrameHeader(&prefix, uint64(len(p)))
	if _, err := w.stream.Write(prefix[:n]); err != nil {
		return 0, err
	}
	return w.stream.Write(p)
}

func (w *xResponseWriter) Flush() { _ = w.FlushError() }
func (w *xResponseWriter) FlushError() error {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.err != nil {
		return w.err
	}
	return w.stream.flush()
}

func (w *xResponseWriter) SetReadDeadline(deadline time.Time) error {
	if w.readCancel != nil {
		w.readCancel()
	}
	ctx := w.ctx
	if deadline.IsZero() {
		w.readCancel = nil
	} else {
		ctx, w.readCancel = context.WithDeadline(w.ctx, deadline)
	}
	w.stream.setReadContext(ctx)
	return nil
}

func (w *xResponseWriter) SetWriteDeadline(deadline time.Time) error {
	if w.writeCancel != nil {
		w.writeCancel()
	}
	ctx := w.ctx
	if deadline.IsZero() {
		w.writeCancel = nil
	} else {
		ctx, w.writeCancel = context.WithDeadline(w.ctx, deadline)
	}
	w.stream.setWriteContext(ctx)
	return nil
}

var _ http.ResponseWriter = (*xResponseWriter)(nil)
var _ http.Flusher = (*xResponseWriter)(nil)
