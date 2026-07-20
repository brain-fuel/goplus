package websocket

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/flate"
)

const perMessageDeflate = "permessage-deflate"

var ErrInvalidExtension = errors.New("websocket: invalid extension negotiation")

// CompressionOptions enables RFC 7692 permessage-deflate. Window-bit values
// are zero (do not advertise) or 8..15. This implementation always negotiates
// no-context-takeover in both directions, making each message independently
// decodable and bounding memory retained between messages.
type CompressionOptions struct {
	ClientMaxWindowBits        int
	ServerMaxWindowBits        int
	AllowClientContextTakeover bool
	AllowServerContextTakeover bool
}

type compressionSettings struct {
	writeWindow int
	readWindow  int
	readContext bool
}

func (o CompressionOptions) validate() error {
	for _, bits := range []int{o.ClientMaxWindowBits, o.ServerMaxWindowBits} {
		if bits != 0 && (bits < 8 || bits > 15) {
			return fmt.Errorf("%w: window bits must be 8..15", ErrInvalidExtension)
		}
	}
	return nil
}

type extension struct {
	name   string
	params map[string]string
}

func splitHeaderList(header string) ([]string, error) {
	var parts []string
	start, quoted, escaped := 0, false, false
	for i, r := range header {
		switch {
		case escaped:
			escaped = false
		case quoted && r == '\\':
			escaped = true
		case r == '"':
			quoted = !quoted
		case r == ',' && !quoted:
			parts = append(parts, strings.TrimSpace(header[start:i]))
			start = i + 1
		}
	}
	if quoted || escaped {
		return nil, ErrInvalidExtension
	}
	parts = append(parts, strings.TrimSpace(header[start:]))
	return parts, nil
}

func parseExtensions(header string) ([]extension, error) {
	if strings.TrimSpace(header) == "" {
		return nil, nil
	}
	parts, err := splitHeaderList(header)
	if err != nil {
		return nil, err
	}
	exts := make([]extension, 0, len(parts))
	for _, part := range parts {
		fields, err := splitDelimited(part, ';')
		if err != nil || len(fields) == 0 || strings.TrimSpace(fields[0]) == "" {
			return nil, ErrInvalidExtension
		}
		params := make(map[string]string, len(fields)-1)
		for _, field := range fields[1:] {
			field = strings.TrimSpace(field)
			if field == "" {
				return nil, ErrInvalidExtension
			}
			key, value, hasValue := strings.Cut(field, "=")
			key = strings.ToLower(strings.TrimSpace(key))
			if key == "" {
				return nil, ErrInvalidExtension
			}
			if _, duplicate := params[key]; duplicate {
				return nil, ErrInvalidExtension
			}
			value = strings.TrimSpace(value)
			if hasValue && strings.HasPrefix(value, "\"") {
				value, err = strconv.Unquote(value)
				if err != nil {
					return nil, ErrInvalidExtension
				}
			} else if !hasValue {
				value = ""
			}
			params[key] = value
		}
		exts = append(exts, extension{name: strings.ToLower(strings.TrimSpace(fields[0])), params: params})
	}
	return exts, nil
}

func splitDelimited(value string, delimiter rune) ([]string, error) {
	var parts []string
	start, quoted, escaped := 0, false, false
	for i, r := range value {
		switch {
		case escaped:
			escaped = false
		case quoted && r == '\\':
			escaped = true
		case r == '"':
			quoted = !quoted
		case r == delimiter && !quoted:
			parts = append(parts, value[start:i])
			start = i + 1
		}
	}
	if quoted || escaped {
		return nil, ErrInvalidExtension
	}
	return append(parts, value[start:]), nil
}

func windowBits(value string, allowEmpty bool) (int, error) {
	if value == "" && allowEmpty {
		return 0, nil
	}
	bits, err := strconv.Atoi(value)
	if err != nil || bits < 8 || bits > 15 {
		return 0, ErrInvalidExtension
	}
	return bits, nil
}

func compressionOffer(o CompressionOptions) (string, error) {
	if err := o.validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("permessage-deflate")
	if !o.AllowServerContextTakeover {
		b.WriteString("; server_no_context_takeover")
	}
	if !o.AllowClientContextTakeover {
		b.WriteString("; client_no_context_takeover")
	}
	if o.ServerMaxWindowBits != 0 {
		fmt.Fprintf(&b, "; server_max_window_bits=%d", o.ServerMaxWindowBits)
	}
	if o.ClientMaxWindowBits != 0 {
		fmt.Fprintf(&b, "; client_max_window_bits=%d", o.ClientMaxWindowBits)
	}
	return b.String(), nil
}

// acceptCompressionResponse validates a server response to our offer and
// returns the window used for outgoing client messages.
func acceptCompressionResponse(header string, offered CompressionOptions) (bool, compressionSettings, error) {
	exts, err := parseExtensions(header)
	if err != nil {
		return false, compressionSettings{}, err
	}
	if len(exts) == 0 {
		return false, compressionSettings{}, nil
	}
	if len(exts) != 1 || exts[0].name != perMessageDeflate {
		return false, compressionSettings{}, ErrInvalidExtension
	}
	p := exts[0].params
	for name := range p {
		switch name {
		case "server_no_context_takeover", "client_no_context_takeover", "server_max_window_bits", "client_max_window_bits":
		default:
			return false, compressionSettings{}, ErrInvalidExtension
		}
	}
	_, serverNoContext := p["server_no_context_takeover"]
	_, clientNoContext := p["client_no_context_takeover"]
	if !serverNoContext && !offered.AllowServerContextTakeover {
		return false, compressionSettings{}, ErrInvalidExtension
	}
	if clientNoContext && offered.AllowClientContextTakeover {
		return false, compressionSettings{}, ErrInvalidExtension
	}
	if p["server_no_context_takeover"] != "" || p["client_no_context_takeover"] != "" {
		return false, compressionSettings{}, ErrInvalidExtension
	}
	settings := compressionSettings{writeWindow: 15, readWindow: 15, readContext: !serverNoContext}
	if raw, ok := p["client_max_window_bits"]; ok {
		if offered.ClientMaxWindowBits == 0 {
			return false, compressionSettings{}, ErrInvalidExtension
		}
		settings.writeWindow, err = windowBits(raw, false)
		if err != nil || settings.writeWindow > offered.ClientMaxWindowBits {
			return false, compressionSettings{}, ErrInvalidExtension
		}
	}
	if raw, ok := p["server_max_window_bits"]; ok {
		bits, parseErr := windowBits(raw, false)
		if parseErr != nil || offered.ServerMaxWindowBits == 0 || bits > offered.ServerMaxWindowBits {
			return false, compressionSettings{}, ErrInvalidExtension
		}
		settings.readWindow = bits
	}
	return true, settings, nil
}

// negotiateCompression selects a safe no-context-takeover offer and returns
// its response value and the window used for outgoing server messages.
func negotiateCompression(header string, configured CompressionOptions) (string, compressionSettings, error) {
	if err := configured.validate(); err != nil {
		return "", compressionSettings{}, err
	}
	exts, err := parseExtensions(header)
	if err != nil {
		return "", compressionSettings{}, err
	}
	for _, ext := range exts {
		if ext.name != perMessageDeflate {
			continue
		}
		p := ext.params
		valid := true
		for name := range p {
			switch name {
			case "server_no_context_takeover", "client_no_context_takeover", "server_max_window_bits", "client_max_window_bits":
			default:
				valid = false
			}
		}
		if !valid || p["server_no_context_takeover"] != "" || p["client_no_context_takeover"] != "" {
			continue
		}
		serverLimit := 15
		if raw, ok := p["server_max_window_bits"]; ok {
			serverLimit, err = windowBits(raw, false)
			if err != nil {
				continue
			}
		}
		writeBits := serverLimit
		if configured.ServerMaxWindowBits != 0 && configured.ServerMaxWindowBits < writeBits {
			writeBits = configured.ServerMaxWindowBits
		}
		settings := compressionSettings{writeWindow: writeBits, readWindow: 15}
		response := "permessage-deflate"
		if !configured.AllowServerContextTakeover {
			response += "; server_no_context_takeover"
		}
		_, offeredClientNoContext := p["client_no_context_takeover"]
		if !configured.AllowClientContextTakeover && offeredClientNoContext {
			response += "; client_no_context_takeover"
		} else {
			settings.readContext = true
		}
		if _, ok := p["server_max_window_bits"]; ok {
			response += fmt.Sprintf("; server_max_window_bits=%d", writeBits)
		}
		if raw, ok := p["client_max_window_bits"]; ok {
			clientLimit, parseErr := windowBits(raw, true)
			if parseErr != nil {
				continue
			}
			clientBits := configured.ClientMaxWindowBits
			if clientBits == 0 {
				clientBits = 15
			}
			if clientLimit != 0 && clientLimit < clientBits {
				clientBits = clientLimit
			}
			response += fmt.Sprintf("; client_max_window_bits=%d", clientBits)
			settings.readWindow = clientBits
		}
		return response, settings, nil
	}
	return "", compressionSettings{}, nil
}

type deflater struct {
	buffer bytes.Buffer
	writer *flate.Writer
}

var deflaterPools [16]sync.Pool

func acquireDeflater(bits int) (*deflater, error) {
	if bits == 0 {
		bits = 15
	}
	if value := deflaterPools[bits].Get(); value != nil {
		return value.(*deflater), nil
	}
	d := &deflater{}
	w, err := flate.NewWriterWindow(&d.buffer, 1<<bits)
	if err != nil {
		return nil, err
	}
	d.writer = w
	return d, nil
}

func deflateMessage(payload []byte, bits int) ([]byte, error) {
	if bits == 0 {
		bits = 15
	}
	d, err := acquireDeflater(bits)
	if err != nil {
		return nil, err
	}
	d.buffer.Reset()
	d.writer.Reset(&d.buffer)
	_, err = d.writer.Write(payload)
	if err == nil {
		err = d.writer.Flush()
	}
	compressed := append([]byte(nil), d.buffer.Bytes()...)
	d.writer.Reset(io.Discard)
	deflaterPools[bits].Put(d)
	if err != nil {
		return nil, err
	}
	if len(compressed) < 4 || !bytes.Equal(compressed[len(compressed)-4:], []byte{0, 0, 0xff, 0xff}) {
		return nil, ErrInvalidExtension
	}
	return compressed[:len(compressed)-4], nil
}

type messageInflater struct {
	history []byte
	window  int
	context bool
}

func (d *messageInflater) inflate(payload []byte, limit int64) ([]byte, error) {
	src := make([]byte, len(payload)+4)
	copy(src, payload)
	copy(src[len(payload):], []byte{0, 0, 0xff, 0xff})
	var r io.ReadCloser
	if d.context && len(d.history) != 0 {
		r = flate.NewReaderDict(bytes.NewReader(src), d.history)
	} else {
		r = flate.NewReader(bytes.NewReader(src))
	}
	defer r.Close()
	var out []byte
	var err error
	if limit <= 0 {
		out, err = io.ReadAll(r)
	} else {
		out, err = io.ReadAll(io.LimitReader(r, limit+1))
	}
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	if limit > 0 && int64(len(out)) > limit {
		return nil, &failureDetail{cause: ErrMessageTooLarge, limit: limit}
	}
	if d.context {
		window := 1 << d.window
		if len(out) >= window {
			d.history = append(d.history[:0], out[len(out)-window:]...)
		} else {
			keep := window - len(out)
			if len(d.history) > keep {
				d.history = d.history[len(d.history)-keep:]
			}
			history := make([]byte, 0, len(d.history)+len(out))
			history = append(history, d.history...)
			d.history = append(history, out...)
		}
	}
	return out, nil
}

func inflateMessage(payload []byte, limit int64) ([]byte, error) {
	return (&messageInflater{window: 15}).inflate(payload, limit)
}
