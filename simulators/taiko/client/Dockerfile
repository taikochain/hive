# This simulation runs JSON-RPC API tests.
FROM golang:1.18-alpine3.15 as builder
RUN apk add --update gcc musl-dev linux-headers

# Build the simulator executable, from hive repo root.
# See context.txt for docker build context change.
# We use a go.work file to pull in other go modules of the hive repo.

COPY ./go.mod /source/go.mod
COPY ./go.sum /source/go.sum

# Copy hive libs
COPY ./hiveproxy /source/hive/hiveproxy
COPY ./internal /source/hive/internal
COPY ./hivesim /source/hive/hivesim

# Copy taiko libs
COPY ./taiko /source/taiko

# Copy simulator codes
COPY ./simulators/taiko/client/hive.go.work /source/go.work
COPY ./simulators/taiko/client/* /source

WORKDIR /source/
RUN go build -v .

# Build the simulator run container.
FROM alpine:latest

COPY --from=builder /source/taiko/config.json .
COPY --from=builder /source/main .

ENTRYPOINT ["./main"]
