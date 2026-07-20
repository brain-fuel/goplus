#!/bin/sh
set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
std=$(CDPATH= cd -- "$here/../.." && pwd)
suffix=$$
autobahn_image="docker.io/crossbario/autobahn-testsuite@sha256:519915fb568b04c9383f70a1c405ae3ff44ab9e35835b085239c258b6fac3074"
network="goplus-ws-autobahn-$suffix"
server="goplus-ws-testee-$suffix"
suite="goplus-ws-suite-$suffix"

cleanup() {
  podman rm -f "$suite" "$server" >/dev/null 2>&1 || true
  podman network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

mkdir -p "$here/reports"
podman build -t goplus-websocket-autobahn -f "$here/websocket.Dockerfile" "$std"
podman network create "$network" >/dev/null
podman run -d --name "$server" --network "$network" --network-alias websocket localhost/goplus-websocket-autobahn:latest >/dev/null
podman run --name "$suite" --network "$network" \
  -v "$here/fuzzingclient.json:/config/fuzzingclient.json:ro" \
  -v "$here/reports:/reports" \
  "$autobahn_image" \
  wstest -m fuzzingclient -s /config/fuzzingclient.json
go run "$here/verify" -reports "$here/reports" -agent goplus-std-websocket -expect 517
