# Go+ HTTP transport capabilities

`goforge.dev/goplus/std/http` complements `net/http` with one client and server
surface for HTTP/3, HTTP/2, and HTTP/1.1.

```go
transport := new(http.Transport)
client := &nethttp.Client{Transport: transport}

// The first request uses HTTP/2 or HTTP/1.1 and learns an h3 Alt-Svc.
// Later requests use HTTP/3 while the advertisement remains valid.
response, err := client.Get("https://example.test/assays")
```

`Transport` implements `net/http.RoundTripper`. In `Auto` mode it does not
blindly probe UDP: it learns HTTP/3 origin capability from Alt-Svc, including
alternate UDP ports, and preserves the original authority and TLS server name.
If HTTP/3 becomes unavailable, replayable requests retry through Go's HTTP/2
or HTTP/1.1 transport. Unreplayable request bodies are never consumed by a
speculative attempt. Certificate verification and caller cancellation errors
are not treated as downgrade signals.

A failed H3 alternative is suppressed for five minutes, preventing an
unchanged fallback Alt-Svc response from causing another failed UDP probe on
every request. A newly advertised authority can be tried immediately, and
explicit prior knowledge overrides the cooldown.

Use `PriorKnowledge` for origins whose HTTP/3 capability is configured out of
band, `HTTP3Only` when fallback is forbidden, or `HTTP2Or1Only` to disable
QUIC.

`Server` serves one `net/http.Handler` over native HTTP/3 on QUIC/UDP and
TLS-based HTTP/2 and HTTP/1.1 on TCP. It publishes the correct Alt-Svc port on
TCP responses:

```go
server := &http.Server{Handler: mux, TLSConfig: tlsConfig}
err := server.Serve(tcpListener, udpPacketConn)
```

The TCP side enables HTTP/2 and HTTP/1.1. `Shutdown` coordinates shutdown of
both protocol families. `NativeHTTP3` and `NativeQUICConfig` customize the native
server; setting `HTTP3` explicitly opts into a quic-go reference server.
