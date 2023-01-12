#!/bin/sh

set -e

workdir=$(
    cd $(dirname $0)
    pwd
)

L1ContainerName=taiko-l1
L2ContainerName=taiko-l2

L1_CHAIN_ID=$(jq -r .l1_network_id ${workdir}/../taiko/config.json)
L1_CLIQUE_PERIOD=$(jq -r .l1_clique_period ${workdir}/../taiko/config.json)
HIVE_TAIKO_L2_CHAIN_ID=$(jq -r .l2_network_id ${workdir}/../taiko/config.json)

startL1Container() {
    echo "Run container"
    docker container rm -f ${L1ContainerName}
    containerID=$(
        docker run \
            -d \
            --name ${L1ContainerName} \
            -e HIVE_TAIKO_L1_CHAIN_ID=${L1_CHAIN_ID} \
            -e HIVE_CLIQUE_PERIOD=${L1_CLIQUE_PERIOD} \
            -v ${workdir}/genesis.json:/tmp/genesis.json \
            -v ${workdir}/start-l1.sh:/start.sh \
            -p 18545:8545 \
            --entrypoint "/start.sh" \
            ethereum/client-go:latest
    )
}

getL2TaikoAddress() {
    docker container rm -f ${L2ContainerName}
    docker run \
        -d \
        --name ${L2ContainerName} \
        gcr.io/evmchain/taiko-geth:taiko

    docker cp ${L2ContainerName}:/deployments/mainnet.json mainnet.json
    docker container rm -f ${L2ContainerName}
    HIVE_TAIKO_L2_ROLLUP_ADDRESS=$(jq -r 'with_entries(select(.value.contractName=="TaikoL2"))|keys|.[0]' mainnet.json)
}

wait() {
    while ! curl \
        --fail \
        --silent \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":0,"method":"eth_chainId","params":[]}' \
        localhost:28545 >/dev/null; do
        sleep 1
    done
}

getL2GenesisBlockHash() {
    docker container rm -f ${L2ContainerName}
    image_name="taiko-geth:tmp"
    cd ${workdir}/../clients/taiko-geth && docker build -t ${image_name} . && cd -
    docker run \
        -d \
        -e HIVE_NETWORK_ID=${HIVE_TAIKO_L2_CHAIN_ID} \
        -e HIVE_TAIKO_JWT_SECRET=$(jq -r .jwt_secret ${workdir}/../taiko/config.json) \
        -p 28545:8545 \
        --name ${L2ContainerName} \
        ${image_name}
    wait
    HIVE_TAIKO_L2_GENESIS_BLOCK_HASH=$(
        curl \
            --silent \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","id":0,"method":"eth_getBlockByNumber","params":["0x0", false]}' \
            localhost:28545 | jq -r .result.hash
    )
    docker container rm -f ${L2ContainerName}
    docker image rm ${image_name}
}

deployProtocol() {
    echo "compile protocol"

    if [ ! -d "taiko-mono" ]; then
        git clone --depth=1 https://github.com/taikoxyz/taiko-mono.git
    fi

    cp ${workdir}/LibSharedConfig.sol taiko-mono/packages/protocol/contracts/libs/LibSharedConfig.sol

    cd taiko-mono/packages/protocol

    pnpm install && K_CHAIN_ID=${HIVE_TAIKO_L2_CHAIN_ID} pnpm compile

    echo "Start deploying contact on ${containerID}"

    HIVE_TAIKO_MAINNET_URL="http://127.0.0.1:18545"
    HIVE_TAIKO_PRIVATE_KEY=$(jq -r .deploy_private_key ${workdir}/../taiko/config.json)
    HIVE_TAIKO_L1_DEPLOYER_ADDRESS=$(jq -r .deploy_address ${workdir}/../taiko/config.json)
    HIVE_TAIKO_L2_ROLLUP_ADDRESS=""
    HIVE_TAIKO_L2_GENESIS_BLOCK_HASH=""

    getL2GenesisBlockHash
    getL2TaikoAddress

    echo HIVE_TAIKO_MAINNET_URL: "$HIVE_TAIKO_MAINNET_URL"
    echo HIVE_TAIKO_PRIVATE_KEY: "$HIVE_TAIKO_PRIVATE_KEY"

    export MAINNET_URL="$HIVE_TAIKO_MAINNET_URL"
    export PRIVATE_KEY="$HIVE_TAIKO_PRIVATE_KEY"

    FLAGS="--network mainnet"
    FLAGS="$FLAGS --dao-vault $HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
    FLAGS="$FLAGS --team-vault $HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
    FLAGS="$FLAGS --l2-genesis-block-hash $HIVE_TAIKO_L2_GENESIS_BLOCK_HASH"
    FLAGS="$FLAGS --l2-chain-id $HIVE_TAIKO_L2_CHAIN_ID"
    FLAGS="$FLAGS --taiko-l2 $HIVE_TAIKO_L2_ROLLUP_ADDRESS"
    FLAGS="$FLAGS --confirmations 1"

    echo "Deploy L1 rollup contacts with flags $FLAGS"

    LOG_LEVEL=debug npx hardhat deploy_L1 $FLAGS

    docker cp deployments/mainnet_L1.json "${L1ContainerName}:/mainnet_L1.json"

    echo "Success to deploy contact on ${containerID}"
}

buildL1Image() {
    docker commit -m $(whoami) -m "taiko-l1-image" ${containerID} taiko-l1:local
    docker container rm -f ${L1ContainerName}
    echo "Success to build taiko-l1 image"
}

startL1Container
deployProtocol
buildL1Image
