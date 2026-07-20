#!/bin/sh
set -eu
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mkdir -p "$here/reports"
docker compose -f "$here/docker-compose.yml" up --build --abort-on-container-exit --exit-code-from autobahn
go run "$here/verify" -reports "$here/reports" -agent goplus-std-websocket -expect 517
