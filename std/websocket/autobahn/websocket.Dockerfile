FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /autobahn-server ./websocket/cmd/autobahn-server

FROM alpine:3.22
COPY --from=build /autobahn-server /usr/local/bin/autobahn-server
ENTRYPOINT ["autobahn-server", "-listen", ":9001"]
