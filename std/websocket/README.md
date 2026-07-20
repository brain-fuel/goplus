# Go+ WebSocket

`goforge.dev/goplus/std/websocket` implements RFC 6455 and RFC 7692 for Go+
and Go. The public message vocabulary is a closed Go+ sum type; framing hot
paths remain allocation-free Go so both languages use the same wire code.

## Server

```go
conn, protocol, err := websocket.Upgrade(w, r, websocket.UpgradeOptions{
    Protocols:   []string{"assay.v1"},
    Compression: &websocket.CompressionOptions{},
})
if err != nil { return }
defer conn.Close()

message, err := conn.ReadMessage()
if err == nil {
    err = conn.WriteMessage(message)
}
_ = protocol
```

`Upgrade` does not impose an origin policy. Browser-facing endpoints should set
`CheckOrigin: websocket.SameOrigin`; non-browser clients without an `Origin`
header remain accepted by that helper.

## Client

```go
conn, response, err := websocket.Dial(ctx, "wss://example.test/socket",
    websocket.DialOptions{
        Protocols: []string{"assay.v1"},
        Compression: &websocket.CompressionOptions{
            ClientMaxWindowBits: 15,
        },
    })
if err != nil { return err }
defer conn.Close()
_ = response
return conn.WriteMessage(websocket.TextMessage{Payload: []byte("hello")})
```

`WriteMessage` preserves caller-owned payload bytes. `WriteMessageOwned`
transfers ownership and avoids the defensive masking copy on clients. One
reader and one writer may operate concurrently; writes are serialized.

## Conformance and performance

The complete Autobahn 25.10.1 server and client gates require Podman. The
runner pins the suite image by digest so the 517-case contract cannot drift:

```sh
./websocket/autobahn/run-podman.sh
./websocket/autobahn/run-client-podman.sh
```

Docker Compose users can run `websocket/autobahn/run.sh`. Reports are written
under `websocket/autobahn/reports/` and the verifier fails on any non-passing
required case.

The comparative gobwas/ws performance contract is:

```sh
go run ./websocket/cmd/benchgate
```

Protocol, compression, conformance, and performance requirements live in
`features/*.feature`; normal tests execute every non-benchmark scenario.
