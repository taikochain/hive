HIVEFLAGS=--client=taiko-l1,taiko-geth,taiko-client
HIVEFLAGS+=--loglevel 4
HIVEFLAGS+=--docker.output
HIVEFLAGS+=--docker.nocache taiko
build:
	@go build . && go build -o hiveview cmd/hiveview/*.go

clean:
	@rm -rf hive hiveview taiko-mono

image:
	@./taiko-image/build-taiko-l1.sh

testops: image build
	@echo "$(shell date) Starting taiko/ops simulation"
	./hive --sim=taiko/ops ${HIVEFLAGS}

testrpc: image build
	@echo "$(shell date) Starting taiko/rpc simulation"
	./hive --sim=taiko/rpc ${HIVEFLAGS}

test: image build
	@echo "$(shell date '+%c') Starting taiko simulation"
	./hive --sim=taiko ${HIVEFLAGS}

.PHONY: build \
		image \
		clean \
		test \
		testops \
		testrpc

