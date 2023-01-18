tmp_dir=/mnt/disks/data/tmp

HIVEFLAGS=--client=taiko-l1,taiko-geth,taiko-client
HIVEFLAGS+=--loglevel 4
HIVEFLAGS+=--docker.output
HIVEFLAGS+=--docker.nocache taiko
ifneq (${RESULTS_DIR},"")
	HIVEFLAGS+=--results-root ${RESULTS_DIR}
endif


build:
	@go build . && go build -o hiveview cmd/hiveview/*.go

clean:
	@sh clean.sh${tmp_dir}/taiko-mono ${tmp_dir}/taiko-client

image:
	@./taiko-image/build-l1-image.sh
	@./taiko-image/build-client-image.sh

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

