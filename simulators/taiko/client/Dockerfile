# This simulation runs JSON-RPC API tests.
FROM golang:1.18-alpine3.15 as builder
RUN apk add --update gcc musl-dev linux-headers

# Build the simulator executable, from hive repo root.
# See context.txt for docker build context change.
# We use a go.work file to pull in other go modules of the hive repo.
COPY ./simulators/taiko/client/hive.go.work /source/go.work
COPY ./taiko /source/taiko
COPY ./simulators/taiko/client /source/client

COPY ./go.mod /source/hive/go.mod
COPY ./go.sum /source/hive/go.sum
COPY ./hiveproxy /source/hive/hiveproxy
COPY ./internal /source/hive/internal
COPY ./hivesim /source/hive/hivesim

WORKDIR /source/client
RUN go build -v .

# Build the simulator run container.
FROM alpine:latest

COPY --from=builder /source/taiko/config.json .
COPY --from=builder /source/client/client .

ENTRYPOINT ["./client"]
