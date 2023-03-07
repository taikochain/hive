# This simulation runs JSON-RPC API tests.
FROM golang:1.18-alpine3.15 as builder
RUN apk add --update gcc musl-dev linux-headers

# Build the simulator executable, from hive repo root.
# See context.txt for docker build context change.
# We use a go.work file to pull in other go modules of the hive repo.
COPY ./taiko /source/taiko
COPY ./simulators/taiko/rpc /source/simulators/taiko/rpc

WORKDIR /source/simulators/taiko/rpc
RUN go build -v -o ./sim .

# Build the simulator run container.
FROM alpine:latest

COPY --from=builder /source/taiko/config.json .
COPY --from=builder /source/simulators/taiko/rpc/sim .

ENTRYPOINT ["./sim"]
