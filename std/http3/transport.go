package http3

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	refqpack "github.com/quic-go/qpack"
	"github.com/quic-go/quic-go"
	xquic "goforge.dev/goplus/std/internal/quic"
)

// QUICBackend selects the RFC 9000 engine used below HTTP/3.
type QUICBackend uint8

const (
	QUICGo QUICBackend = iota
	// NativeQUIC selects the Go+ owned RFC 9000 engine.
	NativeQUIC
	// XNetQUIC is retained as a compatibility spelling for NativeQUIC.
	// Deprecated: use NativeQUIC.
	XNetQUIC = NativeQUIC
)

// RFC9000Config configures the Go+ owned RFC 9000 transport engine.
type RFC9000Config = xquic.Config

const (
	errorRequestCancelled = 0x10c
	errorMessage          = 0x10e
	errorClosedCritical   = 0x104
	errorFrameUnexpected  = 0x105
	errorFrame            = 0x106
	errorID               = 0x108
	errorSettings         = 0x109

	streamTypeControl      = 0
	streamTypePush         = 1
	streamTypeQPACKEncoder = 2
	streamTypeQPACKDecoder = 3
	frameTypeData          = 0
	frameTypeHeaders       = 1
	frameTypeSettings      = 4
	settingMaxFieldSection = 6
	settingExtendedConnect = 8
	maxFieldSectionSize    = 1 << 20
)

// Transport is a native Go+ HTTP/3 client transport. It uses QUIC v1 for
// transport and the package's stateless QPACK encoder for request fields.
type Transport struct {
	Backend          QUICBackend
	TLSClientConfig  *tls.Config
	QUICConfig       *quic.Config
	NativeQUICConfig *RFC9000Config
	// XQUICConfig is retained for compatibility. NativeQUICConfig takes
	// precedence when both are set.
	// Deprecated: use NativeQUICConfig.
	XQUICConfig *RFC9000Config
	Dial        func(context.Context, string, *tls.Config, *quic.Config) (*quic.Conn, error)
	// MaxConnectionsPerOrigin bounds adaptive connection sharding. Zero uses
	// one connection. Sequential traffic continues to use one connection;
	// additional connections are created only while all existing connections
	// have active requests.
	MaxConnectionsPerOrigin int

	mu         sync.Mutex
	clients    map[string][]*clientConn
	fast       atomic.Pointer[clientConn]
	xendpoints map[string]*xquic.Endpoint
	closed     bool
}

var _ http.RoundTripper = (*Transport)(nil)

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil || req.URL.Scheme != "https" {
		return nil, errors.New("http3: HTTPS request required")
	}
	if req.Method == "" {
		return nil, errors.New("http3: request method required")
	}
	client, err := t.client(req.Context(), authorityAddr(req.URL))
	if err != nil {
		return nil, err
	}
	defer client.active.Add(-1)
	return client.roundTrip(req)
}

func authorityAddr(u *url.URL) string {
	if _, _, err := net.SplitHostPort(u.Host); err == nil {
		return u.Host
	}
	return net.JoinHostPort(u.Hostname(), "443")
}

func (t *Transport) client(ctx context.Context, address string) (*clientConn, error) {
	maxConnections := t.MaxConnectionsPerOrigin
	if maxConnections <= 0 {
		maxConnections = 1
	}
	if maxConnections == 1 {
		if client := t.fast.Load(); client != nil && client.authority == address && client.conn.Context().Err() == nil {
			client.active.Add(1)
			return client, nil
		}
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, errors.New("http3: transport closed")
	}
	pool := t.clients[address]
	var available *clientConn
	for _, client := range pool {
		if client.conn.Context().Err() != nil {
			continue
		}
		if available == nil || client.active.Load() < available.active.Load() {
			available = client
		}
	}
	if available != nil && (available.active.Load() == 0 || len(pool) >= maxConnections) {
		available.active.Add(1)
		t.fast.Store(available)
		t.mu.Unlock()
		return available, nil
	}
	tlsConfig := t.TLSClientConfig
	if tlsConfig == nil {
		tlsConfig = new(tls.Config)
	} else {
		tlsConfig = tlsConfig.Clone()
	}
	if tlsConfig.ServerName == "" {
		host, _, _ := net.SplitHostPort(address)
		tlsConfig.ServerName = host
	}
	if tlsConfig.MinVersion < tls.VersionTLS13 {
		tlsConfig.MinVersion = tls.VersionTLS13
	}
	tlsConfig.NextProtos = []string{"h3"}
	backend := t.Backend
	var endpoint *xquic.Endpoint
	if backend == NativeQUIC {
		endpoint = t.xendpoints[address]
		if endpoint == nil {
			var err error
			endpoint, err = newNativeClientEndpoint(address)
			if err != nil {
				t.mu.Unlock()
				return nil, err
			}
			if t.xendpoints == nil {
				t.xendpoints = make(map[string]*xquic.Endpoint)
			}
			t.xendpoints[address] = endpoint
		}
	}
	t.mu.Unlock()
	var conn wireConn
	if backend == NativeQUIC {
		config := t.NativeQUICConfig
		if config == nil {
			config = t.XQUICConfig
		}
		if config == nil {
			config = new(xquic.Config)
		} else {
			config = config.Clone()
		}
		config.TLSConfig = tlsConfig
		rawConn, err := endpoint.Dial(ctx, "udp", address, config)
		if err != nil {
			return nil, err
		}
		conn = newXNetConn(rawConn)
	} else {
		quicConfig := t.QUICConfig
		if quicConfig != nil {
			quicConfig = quicConfig.Clone()
		}
		dial := t.Dial
		if dial == nil {
			dial = quic.DialAddr
		}
		rawConn, err := dial(ctx, address, tlsConfig, quicConfig)
		if err != nil {
			return nil, err
		}
		conn = &quicGoConn{conn: rawConn}
	}
	client, err := newClientConn(ctx, conn)
	if err != nil {
		_ = conn.closeWithError(0x102, err.Error())
		return nil, err
	}
	client.authority = address
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		_ = conn.closeWithError(0x100, "transport closed")
		return nil, errors.New("http3: transport closed")
	}
	if t.clients == nil {
		t.clients = make(map[string][]*clientConn)
	}
	pool = t.clients[address]
	if len(pool) >= maxConnections {
		var existing *clientConn
		for _, candidate := range pool {
			if candidate.conn.Context().Err() == nil && (existing == nil || candidate.active.Load() < existing.active.Load()) {
				existing = candidate
			}
		}
		if existing != nil {
			existing.active.Add(1)
			t.fast.Store(existing)
			t.mu.Unlock()
			_ = conn.closeWithError(0x100, "connection pool full")
			return existing, nil
		}
	}
	client.active.Add(1)
	t.clients[address] = append(pool, client)
	t.fast.Store(client)
	t.mu.Unlock()
	return client, nil
}

// newNativeClientEndpoint binds to the source address selected by the kernel
// for the destination. Besides preserving the correct interface on multihomed
// hosts, a concrete bind avoids per-packet source-address control messages.
func newNativeClientEndpoint(address string) (*xquic.Endpoint, error) {
	remote, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	probe, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return nil, err
	}
	local, ok := probe.LocalAddr().(*net.UDPAddr)
	closeErr := probe.Close()
	if !ok {
		return nil, errors.New("http3: UDP route probe returned a non-UDP address")
	}
	if closeErr != nil {
		return nil, closeErr
	}
	packetConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: local.IP, Zone: local.Zone})
	if err != nil {
		return nil, err
	}
	endpoint, err := xquic.NewEndpoint(packetConn, nil)
	if err != nil {
		_ = packetConn.Close()
		return nil, err
	}
	return endpoint, nil
}

// Close closes all pooled HTTP/3 connections.
func (t *Transport) Close() error {
	t.mu.Lock()
	t.closed = true
	clients := t.clients
	endpoints := t.xendpoints
	t.fast.Store(nil)
	t.clients = nil
	t.xendpoints = nil
	t.mu.Unlock()
	var errs []error
	for _, pool := range clients {
		for _, client := range pool {
			if err := client.conn.closeWithError(0x100, ""); err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, endpoint := range endpoints {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := endpoint.Close(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type clientConn struct {
	conn      wireConn
	authority string
	active    atomic.Int64

	settingsDone chan struct{}
	settingsOnce sync.Once
	settingsErr  error
	extended     bool
	uniMu        sync.Mutex
	controlSeen  bool
	encoderSeen  bool
	decoderSeen  bool
	peerMaxField atomic.Uint64
}

func newClientConn(ctx context.Context, conn wireConn) (*clientConn, error) {
	c := &clientConn{conn: conn, settingsDone: make(chan struct{})}
	c.peerMaxField.Store(^uint64(0))
	control, err := conn.openUni(ctx)
	if err != nil {
		return nil, err
	}
	settingsPayload := appendQUICVarint(nil, settingExtendedConnect)
	settingsPayload = appendQUICVarint(settingsPayload, 1)
	settingsPayload = appendQUICVarint(settingsPayload, settingMaxFieldSection)
	settingsPayload = appendQUICVarint(settingsPayload, maxFieldSectionSize)
	prefix := appendQUICVarint(nil, streamTypeControl)
	prefix, _ = AppendFrameHeader(prefix, frameTypeSettings, uint64(len(settingsPayload)))
	prefix = append(prefix, settingsPayload...)
	if _, err := control.Write(prefix); err != nil {
		return nil, err
	}
	if err := control.flush(); err != nil {
		return nil, err
	}
	for _, streamType := range []uint64{streamTypeQPACKEncoder, streamTypeQPACKDecoder} {
		stream, err := conn.openUni(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := stream.Write(appendQUICVarint(nil, streamType)); err != nil {
			return nil, err
		}
		if err := stream.flush(); err != nil {
			return nil, err
		}
	}
	go c.acceptUniStreams()
	return c, nil
}

func (c *clientConn) acceptUniStreams() {
	for {
		stream, err := c.conn.acceptUni(c.conn.Context())
		if err != nil {
			c.finishSettings(err)
			return
		}
		go c.handleUniStream(stream)
	}
}

func (c *clientConn) handleUniStream(stream io.Reader) {
	reader := &byteReader{Reader: stream}
	streamType, err := readQUICVarint(reader)
	if err != nil {
		c.finishSettings(err)
		return
	}
	if err := c.claimUniStream(streamType); err != nil {
		_ = c.conn.closeWithError(0x103, err.Error())
		c.finishSettings(err)
		return
	}
	if streamType == streamTypePush {
		err := errors.New("http3: push stream received without MAX_PUSH_ID")
		_ = c.conn.closeWithError(0x108, err.Error())
		c.finishSettings(err)
		return
	}
	if streamType != streamTypeControl {
		_, _ = io.Copy(io.Discard, reader)
		if (streamType == streamTypeQPACKEncoder || streamType == streamTypeQPACKDecoder) && c.conn.Context().Err() == nil {
			_ = c.conn.closeWithError(0x104, "critical stream closed")
		}
		return
	}
	frameType, err := readQUICVarint(reader)
	if err != nil {
		c.finishSettings(err)
		return
	}
	length, err := readQUICVarint(reader)
	if err != nil || frameType != frameTypeSettings {
		c.finishSettings(errors.New("http3: peer control stream did not begin with SETTINGS"))
		return
	}
	if length > 8<<10 {
		c.finishSettings(errors.New("http3: SETTINGS frame too large"))
		return
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		c.finishSettings(err)
		return
	}
	settings, err := parseSettings(payload)
	if err != nil {
		c.finishSettings(err)
		return
	}
	c.extended = settings.extended
	c.peerMaxField.Store(settings.maxFieldSection)
	c.finishSettings(nil)
	if code, controlErr := validateControlFrames(reader); controlErr != nil && c.conn.Context().Err() == nil {
		_ = c.conn.closeWithError(code, controlErr.Error())
	}
}

func validateControlFrames(reader *byteReader) (uint64, error) {
	var lastGoAway uint64 = 1<<62 - 1
	for {
		frameType, err := readQUICVarint(reader)
		if err != nil {
			if err == io.EOF {
				return errorClosedCritical, errors.New("http3: control stream closed")
			}
			return errorClosedCritical, err
		}
		length, err := readQUICVarint(reader)
		if err != nil {
			return errorClosedCritical, err
		}
		switch frameType {
		case frameTypeSettings:
			return errorSettings, errors.New("http3: duplicate SETTINGS frame")
		case frameTypeData, frameTypeHeaders, 0x3, 0x5, 0xd:
			return errorFrameUnexpected, fmt.Errorf("http3: frame type %#x forbidden on server control stream", frameType)
		case 0x7: // GOAWAY
			if length == 0 || length > 8 {
				return errorFrame, errors.New("http3: malformed GOAWAY")
			}
			payload := make([]byte, length)
			if _, err := io.ReadFull(reader, payload); err != nil {
				return errorClosedCritical, err
			}
			payloadReader := &byteReader{Reader: bytes.NewReader(payload)}
			identifier, err := readQUICVarint(payloadReader)
			if err != nil || payloadReader.Reader.(*bytes.Reader).Len() != 0 {
				return errorFrame, errors.New("http3: malformed GOAWAY identifier")
			}
			if identifier%4 != 0 || identifier > lastGoAway {
				return errorID, errors.New("http3: invalid GOAWAY identifier")
			}
			lastGoAway = identifier
		default:
			if _, err := io.CopyN(io.Discard, reader, int64(length)); err != nil {
				return errorClosedCritical, err
			}
		}
	}
}

func (c *clientConn) claimUniStream(streamType uint64) error {
	c.uniMu.Lock()
	defer c.uniMu.Unlock()
	var seen *bool
	switch streamType {
	case streamTypeControl:
		seen = &c.controlSeen
	case streamTypeQPACKEncoder:
		seen = &c.encoderSeen
	case streamTypeQPACKDecoder:
		seen = &c.decoderSeen
	default:
		return nil
	}
	if *seen {
		return fmt.Errorf("http3: duplicate critical stream type %d", streamType)
	}
	*seen = true
	return nil
}

type peerSettings struct {
	extended        bool
	maxFieldSection uint64
}

func parseSettings(payload []byte) (peerSettings, error) {
	settings := bytes.NewReader(payload)
	seen := make(map[uint64]struct{})
	result := peerSettings{maxFieldSection: ^uint64(0)}
	for settings.Len() != 0 {
		identifier, err := readQUICVarint(settings)
		if err != nil {
			return peerSettings{}, err
		}
		value, err := readQUICVarint(settings)
		if err != nil {
			return peerSettings{}, err
		}
		if _, duplicate := seen[identifier]; duplicate {
			return peerSettings{}, fmt.Errorf("http3: duplicate setting %d", identifier)
		}
		seen[identifier] = struct{}{}
		switch identifier {
		case 0x2, 0x3, 0x4, 0x5:
			return peerSettings{}, fmt.Errorf("http3: reserved HTTP/2 setting %d", identifier)
		case settingMaxFieldSection:
			result.maxFieldSection = value
		case settingExtendedConnect:
			if value > 1 {
				return peerSettings{}, errors.New("http3: invalid SETTINGS_ENABLE_CONNECT_PROTOCOL")
			}
			result.extended = value == 1
		}
	}
	return result, nil
}

func (c *clientConn) finishSettings(err error) {
	c.settingsOnce.Do(func() {
		c.settingsErr = err
		close(c.settingsDone)
	})
}

func (c *clientConn) roundTrip(req *http.Request) (*http.Response, error) {
	extended := req.Method == http.MethodConnect && req.Proto != "" && req.Proto != "HTTP/1.1"
	if extended {
		select {
		case <-c.settingsDone:
			if c.settingsErr != nil {
				return nil, c.settingsErr
			}
			if !c.extended {
				return nil, errors.New("http3: server did not enable Extended CONNECT")
			}
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	stream, err := c.conn.openStream(req.Context())
	if err != nil {
		return nil, err
	}
	watch := watchStreamContext(req.Context(), stream)
	handedOff := false
	defer func() {
		if !handedOff && watch != nil {
			watch.stop()
		}
	}()
	frame, err := encodeRequestHeadersFrame(req, extended, c.peerMaxField.Load())
	if err != nil {
		if watch != nil {
			watch.stop()
		}
		stream.abortRead(errorMessage)
		stream.abortWrite(errorMessage)
		return nil, err
	}
	if _, err := stream.Write(frame); err != nil {
		return nil, err
	}
	var bodyDone <-chan error
	var requestErr error
	if req.Body == nil || req.Body == http.NoBody {
		requestErr = finishRequest(stream, req.Trailer, c.peerMaxField.Load())
	} else {
		if err := stream.flush(); err != nil {
			return nil, err
		}
		done := make(chan error, 1)
		bodyDone = done
		go func() { done <- writeRequestBody(stream, req.Body, req.Trailer, c.peerMaxField.Load()) }()
	}
	response, responseReader, err := readResponse(stream, req)
	if err != nil {
		if watch != nil {
			watch.stop()
		}
		stream.abortRead(errorRequestCancelled)
		stream.abortWrite(errorRequestCancelled)
		return nil, err
	}
	tlsState := c.conn.tlsState()
	response.TLS = &tlsState
	response.Body = &responseBody{
		stream: stream, reader: responseReader, bodyDone: bodyDone, watch: watch, response: response,
		requestErr: requestErr, expected: response.ContentLength, enforceLength: response.ContentLength >= 0,
	}
	handedOff = true
	return response, nil
}

type streamContextWatch struct {
	done chan struct{}
	once sync.Once
}

func watchStreamContext(ctx context.Context, stream wireStream) *streamContextWatch {
	if !stream.needsContextWatch() {
		return nil
	}
	w := &streamContextWatch{done: make(chan struct{})}
	go func() {
		select {
		case <-ctx.Done():
			stream.abortRead(errorRequestCancelled)
			stream.abortWrite(errorRequestCancelled)
		case <-w.done:
		}
	}()
	return w
}

func (w *streamContextWatch) stop() {
	if w.done != nil {
		w.once.Do(func() { close(w.done) })
	}
}

func requestFields(req *http.Request, extended bool) ([]HeaderField, error) {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	path := req.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	fields := make([]HeaderField, 0, 5+len(req.Header))
	fields = append(fields, HeaderField{Name: ":method", Value: req.Method})
	if req.Method != http.MethodConnect || extended {
		fields = append(fields,
			HeaderField{Name: ":scheme", Value: req.URL.Scheme},
			HeaderField{Name: ":authority", Value: host},
			HeaderField{Name: ":path", Value: path},
		)
	} else {
		fields = append(fields, HeaderField{Name: ":authority", Value: host})
	}
	if extended {
		fields = append(fields, HeaderField{Name: ":protocol", Value: req.Proto})
	}
	for name, values := range req.Header {
		lower := strings.ToLower(name)
		if lower == "host" || lower == "connection" || lower == "upgrade" || lower == "proxy-connection" || lower == "transfer-encoding" || lower == "keep-alive" {
			continue
		}
		if strings.HasPrefix(lower, ":") || !validHeaderName(lower) {
			return nil, fmt.Errorf("http3: invalid header name %q", name)
		}
		if req.Method == http.MethodConnect && lower == "content-length" {
			continue
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return nil, fmt.Errorf("http3: invalid header value for %q", name)
			}
			if lower == "te" && !strings.EqualFold(strings.TrimSpace(value), "trailers") {
				return nil, errors.New("http3: TE header may only contain trailers")
			}
			fields = append(fields, HeaderField{Name: lower, Value: value, Sensitive: lower == "authorization" || lower == "cookie"})
		}
	}
	if req.Method != http.MethodConnect && req.ContentLength > 0 && req.Body != nil && req.Body != http.NoBody && req.Header.Get("Content-Length") == "" {
		fields = append(fields, HeaderField{Name: "content-length", Value: strconv.FormatInt(req.ContentLength, 10)})
	}
	return fields, nil
}

func encodeRequestHeadersFrame(req *http.Request, extended bool, peerLimit uint64) ([]byte, error) {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	path := req.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	frame := make([]byte, 9, 9+256)
	frame = append(frame, 0, 0)
	limit := min(peerLimit, uint64(maxFieldSectionSize))
	var decodedSize uint64
	appendOne := func(name, value string, sensitive bool) error {
		size := uint64(len(name) + len(value) + 32)
		if size > limit-decodedSize {
			return ErrFieldSectionTooLarge
		}
		decodedSize += size
		var err error
		frame, err = appendField(frame, &HeaderField{Name: name, Value: value, Sensitive: sensitive})
		return err
	}
	if err := appendOne(":method", req.Method, false); err != nil {
		return nil, err
	}
	if req.Method != http.MethodConnect || extended {
		for _, field := range [...]HeaderField{
			{Name: ":scheme", Value: req.URL.Scheme},
			{Name: ":authority", Value: host},
			{Name: ":path", Value: path},
		} {
			if err := appendOne(field.Name, field.Value, false); err != nil {
				return nil, err
			}
		}
	} else if err := appendOne(":authority", host, false); err != nil {
		return nil, err
	}
	if extended {
		if err := appendOne(":protocol", req.Proto, false); err != nil {
			return nil, err
		}
	}
	for name, values := range req.Header {
		lower := strings.ToLower(name)
		if lower == "host" || lower == "connection" || lower == "upgrade" || lower == "proxy-connection" || lower == "transfer-encoding" || lower == "keep-alive" {
			continue
		}
		if strings.HasPrefix(lower, ":") || !validHeaderName(lower) {
			return nil, fmt.Errorf("http3: invalid header name %q", name)
		}
		if req.Method == http.MethodConnect && lower == "content-length" {
			continue
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return nil, fmt.Errorf("http3: invalid header value for %q", name)
			}
			if lower == "te" && !strings.EqualFold(strings.TrimSpace(value), "trailers") {
				return nil, errors.New("http3: TE header may only contain trailers")
			}
			if err := appendOne(lower, value, lower == "authorization" || lower == "cookie"); err != nil {
				return nil, err
			}
		}
	}
	if req.Method != http.MethodConnect && req.ContentLength > 0 && req.Body != nil && req.Body != http.NoBody && req.Header.Get("Content-Length") == "" {
		if err := appendOne("content-length", strconv.FormatInt(req.ContentLength, 10), false); err != nil {
			return nil, err
		}
	}
	return finishHeadersFrame(frame), nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := range len(name) {
		c := name[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))) {
			return false
		}
	}
	return true
}

func writeRequestBody(stream wireStream, body io.ReadCloser, trailer http.Header, peerLimit uint64) error {
	defer body.Close()
	buffer := make([]byte, 32<<10)
	var header [9]byte
	for {
		n, err := body.Read(buffer)
		if n != 0 {
			headerLength := EncodeDataFrameHeader(&header, uint64(n))
			if _, writeErr := stream.Write(header[:headerLength]); writeErr != nil {
				return writeErr
			}
			if _, writeErr := stream.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if flushErr := stream.flush(); flushErr != nil {
				return flushErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return finishRequest(stream, trailer, peerLimit)
			}
			return err
		}
	}
}

func finishRequest(stream wireStream, trailer http.Header, peerLimit uint64) error {
	if len(trailer) != 0 {
		fields := make([]HeaderField, 0, len(trailer))
		for name, values := range trailer {
			lower := strings.ToLower(name)
			if strings.HasPrefix(lower, ":") || !validHeaderName(lower) || connectionSpecificField(lower) {
				return fmt.Errorf("http3: invalid request trailer %q", name)
			}
			for _, value := range values {
				if strings.ContainsAny(value, "\r\n") {
					return fmt.Errorf("http3: invalid request trailer value for %q", name)
				}
				fields = append(fields, HeaderField{Name: lower, Value: value})
			}
		}
		frame, err := encodeHeadersFrameLimit(fields, peerLimit)
		if err != nil {
			return err
		}
		if _, err := stream.Write(frame); err != nil {
			return err
		}
		if err := stream.flush(); err != nil {
			return err
		}
	}
	return stream.Close()
}

func readResponse(stream wireStream, req *http.Request) (*http.Response, *byteReader, error) {
	reader := &byteReader{Reader: stream}
	for {
		frameType, err := readQUICVarint(reader)
		if err != nil {
			return nil, nil, err
		}
		length, err := readQUICVarint(reader)
		if err != nil {
			return nil, nil, err
		}
		if frameType != frameTypeHeaders {
			return nil, nil, errors.New("http3: response did not begin with HEADERS")
		}
		fields, err := readFieldSection(reader, length)
		if err != nil {
			return nil, nil, err
		}
		response, err := responseFromFields(fields, req)
		if err != nil {
			return nil, nil, err
		}
		if response.StatusCode >= 200 {
			return response, reader, nil
		}
		if response.StatusCode == http.StatusSwitchingProtocols {
			return nil, nil, errors.New("http3: status 101 is invalid in HTTP/3")
		}
		if trace := httptrace.ContextClientTrace(req.Context()); trace != nil && trace.Got1xxResponse != nil {
			if err := trace.Got1xxResponse(response.StatusCode, textproto.MIMEHeader(response.Header)); err != nil {
				return nil, nil, err
			}
		}
	}
}

func readFieldSection(reader io.Reader, length uint64) ([]refqpack.HeaderField, error) {
	if length > 8<<20 {
		return nil, errors.New("http3: header section too large")
	}
	block := make([]byte, length)
	if _, err := io.ReadFull(reader, block); err != nil {
		return nil, err
	}
	return decodeFieldSection(block)
}

func responseFromFields(fields []refqpack.HeaderField, req *http.Request) (*http.Response, error) {
	response := &http.Response{Proto: "HTTP/3.0", ProtoMajor: 3, Header: make(http.Header), Request: req}
	seenStatus, seenRegular := false, false
	for _, field := range fields {
		if field.Name == ":status" {
			if seenStatus || seenRegular || len(field.Value) != 3 {
				return nil, errors.New("http3: invalid response status")
			}
			code, err := strconv.Atoi(field.Value)
			if err != nil || code < 100 {
				return nil, errors.New("http3: invalid response status")
			}
			response.StatusCode = code
			seenStatus = true
			continue
		}
		if strings.HasPrefix(field.Name, ":") {
			return nil, errors.New("http3: invalid response pseudo-header")
		}
		seenRegular = true
		if field.Name != strings.ToLower(field.Name) || !validHeaderName(field.Name) || strings.ContainsAny(field.Value, "\r\n") {
			return nil, errors.New("http3: invalid response field")
		}
		if connectionSpecificField(field.Name) {
			return nil, fmt.Errorf("http3: forbidden response field %q", field.Name)
		}
		response.Header.Add(textproto.CanonicalMIMEHeaderKey(field.Name), field.Value)
	}
	if response.StatusCode == 0 {
		return nil, errors.New("http3: missing response status")
	}
	response.Status = strconv.Itoa(response.StatusCode) + " " + http.StatusText(response.StatusCode)
	response.ContentLength = -1
	if raw := response.Header.Get("Content-Length"); raw != "" && !(req.Method == http.MethodConnect && response.StatusCode >= 200 && response.StatusCode < 300) {
		contentLength, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || contentLength < 0 {
			return nil, errors.New("http3: invalid content length")
		}
		response.ContentLength = contentLength
	}
	if req.Method == http.MethodHead || response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusNotModified {
		response.ContentLength = 0
	}
	return response, nil
}

func connectionSpecificField(name string) bool {
	switch name {
	case "connection", "keep-alive", "proxy-connection", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

type responseBody struct {
	stream        wireStream
	reader        *byteReader
	remaining     uint64
	bodyDone      <-chan error
	requestErr    error
	watch         *streamContextWatch
	response      *http.Response
	trailers      bool
	expected      int64
	read          int64
	enforceLength bool
	closed        bool
}

func (b *responseBody) Read(p []byte) (int, error) {
	if b.closed {
		return 0, http.ErrBodyReadAfterClose
	}
	if b.reader == nil {
		b.reader = &byteReader{Reader: b.stream}
	}
	for b.remaining == 0 {
		frameType, err := readQUICVarint(b.reader)
		if err != nil {
			return 0, b.endError(err)
		}
		length, err := readQUICVarint(b.reader)
		if err != nil {
			return 0, err
		}
		if frameType == frameTypeData {
			if b.trailers {
				return 0, errors.New("http3: DATA frame after trailers")
			}
			b.remaining = length
			if b.enforceLength && length > uint64(b.expected-b.read) {
				return 0, errors.New("http3: response body exceeds content length")
			}
			if length == 0 {
				continue
			}
			break
		}
		if frameType == frameTypeHeaders {
			if b.trailers {
				return 0, errors.New("http3: multiple trailer sections")
			}
			fields, err := readFieldSection(b.reader, length)
			if err != nil {
				return 0, err
			}
			if b.response.Trailer == nil {
				b.response.Trailer = make(http.Header)
			}
			for _, field := range fields {
				if strings.HasPrefix(field.Name, ":") || field.Name != strings.ToLower(field.Name) ||
					!validHeaderName(field.Name) || connectionSpecificField(field.Name) || strings.ContainsAny(field.Value, "\r\n") {
					return 0, errors.New("http3: invalid trailer field")
				}
				b.response.Trailer.Add(textproto.CanonicalMIMEHeaderKey(field.Name), field.Value)
			}
			b.trailers = true
			continue
		}
		if _, err := io.CopyN(io.Discard, b.reader, int64(length)); err != nil {
			return 0, err
		}
	}
	if uint64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.reader.Read(p)
	b.remaining -= uint64(n)
	b.read += int64(n)
	err = b.endError(err)
	if err != nil && b.watch != nil {
		b.watch.stop()
	}
	return n, err
}

func (b *responseBody) endError(err error) error {
	if err != io.EOF {
		return err
	}
	if b.watch != nil {
		b.watch.stop()
	}
	if b.enforceLength && b.read < b.expected {
		return io.ErrUnexpectedEOF
	}
	if b.requestErr != nil {
		return b.requestErr
	}
	if b.bodyDone != nil {
		select {
		case bodyErr := <-b.bodyDone:
			if bodyErr != nil {
				return bodyErr
			}
		default:
		}
	}
	return io.EOF
}

func (b *responseBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	if b.watch != nil {
		b.watch.stop()
	}
	b.stream.abortRead(errorRequestCancelled)
	return nil
}

type byteReader struct {
	io.Reader
	one [1]byte
}

func (r *byteReader) ReadByte() (byte, error) {
	_, err := io.ReadFull(r.Reader, r.one[:])
	return r.one[0], err
}

func readQUICVarint(reader interface{ ReadByte() (byte, error) }) (uint64, error) {
	first, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	length := 1 << (first >> 6)
	value := uint64(first & 0x3f)
	for range length - 1 {
		next, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		value = value<<8 | uint64(next)
	}
	return value, nil
}
