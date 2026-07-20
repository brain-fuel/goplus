# Go+ HTTP/3

`goforge.dev/goplus/std/http3` provides a native RFC 9114 client, server, and
wire layer. It runs on RFC 9000 QUIC, creates the required control and QPACK
streams, exchanges SETTINGS, supports streaming DATA and trailers in both
directions, and supports RFC 9220 extended CONNECT. The client can use either
quic-go or the Go+ owned RFC 9000 engine; `NativeServer` and `QUICGoServer`
provide native HTTP/3 server data paths over either engine. `XNetServer`
remains as a deprecated alias.

The request field encoder deliberately uses a zero-capacity QPACK strategy.
That removes encoder-stream blocking and permits every request stream to encode
independently. Common static fields use direct QPACK indices; short literals
use the raw representation to avoid paying more CPU for marginal Huffman wire
savings. Long literals use Huffman encoding only when it saves at least eight
bytes. Both endpoints advertise and enforce a 1 MiB decoded field-section
limit, and outgoing headers and trailers honor a smaller limit advertised by
the peer.

The executable comparative gate covers a warmed native client/server round
trip as well as ordinary HTTP and WebSocket field sets. A parallel full-stack
benchmark remains available as diagnostic throughput coverage, while the
release gate uses the reproducible Linux sequential latency workload:

```sh
go run ./http3/cmd/benchgate
```

Every case must remain at least 2× faster than quic-go and allocate no more
bytes. Additional native-stack latency, parallel-throughput, header-heavy, and
payload benchmarks can be run with:

```sh
go test ./http3 -run '^$' -bench '^BenchmarkNativeStack' -benchmem
```

The executable performance contract includes a complete warmed HTTP/3 request
and response through the native client and server, in addition to the isolated
wire-format hot paths. This prevents encoder-only improvements from being
reported as a 2x HTTP/3 implementation result.
