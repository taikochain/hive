# This simulation runs JSON-RPC API tests.
FROM golang:1.18-alpine3.15 as builder
RUN apk add --update gcc musl-dev linux-headers

# Build the simulator executable, from hive repo root.
# See context.txt for docker build context change.
# We use a go.work file to pull in other go modules of the hive repo.
COPY ./simulators/taiko/rpc/hive.go.work /source/go.work
COPY ./taiko /source/taiko
COPY ./simulators/taiko/rpc /source/rpc

WORKDIR /source/rpc
RUN go build -v .

# Build the simulator run container.
FROM alpine:latest

COPY --from=builder /source/rpc/rpc .
COPY ./simulators/taiko/rpc/start.sh /start.sh

RUN chmod +x /start.sh && /start.sh

ENTRYPOINT ["./rpc"]
