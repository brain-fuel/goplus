#!/bin/sh
set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
std=$(CDPATH= cd -- "$here/../.." && pwd)
name="goplus-ws-fuzzingserver-$$"
agent="goplus-std-websocket-client"
autobahn_image="docker.io/crossbario/autobahn-testsuite@sha256:519915fb568b04c9383f70a1c405ae3ff44ab9e35835b085239c258b6fac3074"

cleanup() {
  podman rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

mkdir -p "$here/reports"
podman run -d --name "$name" -p 9001:9001 \
  -v "$here/fuzzingserver.json:/config/fuzzingserver.json:ro" \
  -v "$here/reports:/reports" \
  "$autobahn_image" \
  wstest -m fuzzingserver -s /config/fuzzingserver.json >/dev/null

i=0
until nc -z 127.0.0.1 9001; do
  i=$((i + 1))
  [ "$i" -lt 120 ] || { echo "Autobahn did not start" >&2; exit 2; }
  sleep 0.25
done

(cd "$std" && go run ./websocket/cmd/autobahn-client -url ws://127.0.0.1:9001 -agent "$agent")
(cd "$std" && go run ./websocket/autobahn/verify -reports "$here/reports" -agent "$agent" -expect 517)
