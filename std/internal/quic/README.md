# Go+ internal QUIC engine

This directory contains the RFC 9000 transport engine used by the Go+ HTTP/3
implementation. It is derived from `golang.org/x/net/quic` v0.50.0, commit
`ebddb99633e0fc35d135f62e9400678492c1d3be`, under the BSD license reproduced
in `LICENSE`.

Go+ owns this internal copy so its HTTP/3 stack can tune stream allocation and
scheduling without patching the module cache or forking unrelated x/net
packages. It is not a public API; applications configure it through
`http3.RFC9000Config`.
