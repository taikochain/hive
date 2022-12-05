# This simulation runs JSON-RPC API tests.
FROM golang:1.18-alpine3.15 as builder
RUN apk add --update gcc musl-dev linux-headers

# Build the simulator executable, from hive repo root.
# See context.txt for docker build context change.
# We use a go.work file to pull in other go modules of the hive repo.
COPY ./simulators/taiko/ops/hive.go.work /source/go.work
COPY ./taiko /source/taiko
COPY ./simulators/taiko/ops /source/ops

WORKDIR /source/ops
RUN go build -v .

# Build the simulator run container.
FROM alpine:latest

COPY --from=builder /source/ops/ops .
COPY ./clients/taiko-l1/genesis.json /genesis.json
COPY ./simulators/taiko/rpc/start.sh /start.sh

RUN chmod +x /start.sh && /start.sh

ENTRYPOINT ["./ops"]
