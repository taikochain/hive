tmp_dir=/mnt/disks/data/tmp
result_dir=workspace/logs
ifneq ("${RESULTS_DIR}","")
	result_dir=${RESULTS_DIR}
endif
HIVEFLAGS=--client=taiko-l1,taiko-geth,taiko-client
HIVEFLAGS+=--loglevel 4
HIVEFLAGS+=--docker.output
HIVEFLAGS+=--docker.nocache taiko
HIVEFLAGS+=--results-root ${result_dir}


build:
	@go build . && go build -o hiveview cmd/hiveview/*.go

clean:
	@sh clean.sh${tmp_dir}/taiko-mono ${tmp_dir}/taiko-client

image:
	@./taiko-image/build-l1-image.sh
	@./taiko-image/build-client-image.sh

test-client: image build
	@echo "$(shell date) Starting taiko/client simulation"
	./hive --sim=taiko/client ${HIVEFLAGS}

test-rpc: image build
	@echo "$(shell date) Starting taiko/rpc simulation"
	./hive --sim=taiko/rpc ${HIVEFLAGS}

test: image build
	@echo "$(shell date '+%c') Starting taiko simulation"
	./hive --sim=taiko ${HIVEFLAGS}

.PHONY: build \
		image \
		clean \
		test \
		test-client \
		test-rpc

