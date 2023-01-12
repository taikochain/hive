HIVEFLAGS=--client=taiko-l1,taiko-geth,taiko-protocol,taiko-client
HIVEFLAGS+=--loglevel 4
HIVEFLAGS+=--docker.output

build:
	@go build . && go build -o hiveview cmd/hiveview/*.go

clean:
	@rm -rf hive hiveview

testops: build
	@echo "$(shell date) Starting taiko/ops simulation"
	./hive --sim=taiko/ops ${HIVEFLAGS}

testrpc: build
	@echo "$(shell date) Starting taiko/rpc simulation"
	./hive --sim=taiko/rpc ${HIVEFLAGS}

test: build
	@echo "$(shell date '+%c') Starting taiko simulation"
	# ./hive --sim=taiko ${HIVEFLAGS}

.PHONY: build \
		clean \
		test \
		testops \
		testrpc

