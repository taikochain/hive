#!/usr/bin/env bash

set -e

debug=false

tmp_dir=/mnt/disks/data/tmp

workdir=$(
    cd "$(dirname $0)"
    pwd
)

l1_container_name=taiko-l1
l2_container_name=taiko-l2
taiko_config_file="${workdir}/../taiko/config.json"

l1_network_id=$(jq -r .l1_network_id "${taiko_config_file}")
l1_clique_period=$(jq -r .l1_clique_period "${taiko_config_file}")
l2_network_id=$(jq -r .l2_network_id "${taiko_config_file}")

l2_taiko_addr=""
l2_genesis_hash=""

function get_l2_taiko_Addr() {
    docker container rm -f ${l2_container_name}
    docker run \
        -d \
        --name ${l2_container_name} \
        gcr.io/evmchain/taiko-geth:taiko

    docker cp ${l2_container_name}:/deployments/mainnet.json mainnet.json
    docker container rm -f ${l2_container_name}
    l2_taiko_addr=$(jq -r 'with_entries(select(.value.contractName=="TaikoL2"))|keys|.[0]' mainnet.json)
}

function wait() {
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

function get_l2_genesis_hash() {
    docker container rm -f ${l2_container_name}
    image_name="taiko-geth:tmp"
    cd "${workdir}"/../clients/taiko-geth && docker build -t ${image_name} . && cd -
    docker run \
        -d \
        -e HIVE_NETWORK_ID="${l2_network_id}" \
        -e HIVE_TAIKO_JWT_SECRET="$(jq -r .jwt_secret ${taiko_config_file})" \
        -p 28545:8545 \
        --name ${l2_container_name} \
        ${image_name}
    wait
    l2_genesis_hash=$(
        curl \
            --silent \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","id":0,"method":"eth_getBlockByNumber","params":["0x0", false]}' \
            localhost:28545 | jq -r .result.hash
    )
    docker container rm -f ${l2_container_name}
    docker image rm ${image_name}
}

start_l1_container() {
    echo "Run container"
    docker container rm -f ${l1_container_name}
    containerID=$(
        docker run \
            -d \
            --name ${l1_container_name} \
            -e HIVE_TAIKO_L1_CHAIN_ID="${l1_network_id}" \
            -e HIVE_CLIQUE_PERIOD="${l1_clique_period}" \
            -v "${workdir}/genesis.json":/tmp/genesis.json \
            -v "${workdir}/start-l1.sh":/start.sh \
            -p 18545:8545 \
            --entrypoint "/start.sh" \
            ethereum/client-go:latest
    )
}

deploy_l1_protocol() {
    if [[ "${debug}" == "true" ]]; then
        mono_dir=$(
            cd ../taiko-mono
            pwd
        )
    else
        mono_dir="${tmp_dir}/taiko-mono"
        rm -fr "${mono_dir}"
        git clone --depth=1 https://github.com/taikoxyz/taiko-mono.git ${mono_dir}
    fi

    # change some protocol config for test
    origin_config="${mono_dir}/packages/protocol/contracts/libs/LibSharedConfig.sol"
    changed_config="${workdir}/LibSharedConfig.sol"
    sed -f "${workdir}/LibSharedConfig.sed" "${origin_config}" >"${changed_config}"
    mv "${changed_config}" "${origin_config}"
    # change prove method for test
    cp "${workdir}/LibZKP.sol" "${mono_dir}/packages/protocol/contracts/libs/LibZKP.sol"

    cd "${mono_dir}/packages/protocol"

    if [ ! -f "bin/solc" ]; then
        ./scripts/download_solc.sh
    fi

    pnpm install && K_CHAIN_ID=${l2_network_id} pnpm compile

    echo "Start deploying contact on ${containerID}"

    HIVE_TAIKO_MAINNET_URL="http://127.0.0.1:18545"
    HIVE_TAIKO_PRIVATE_KEY=$(jq -r .deploy_private_key "${taiko_config_file}")
    HIVE_TAIKO_L1_DEPLOYER_ADDRESS=$(jq -r .deploy_address "${taiko_config_file}")

    get_l2_genesis_hash
    get_l2_taiko_Addr

    echo HIVE_TAIKO_MAINNET_URL: "$HIVE_TAIKO_MAINNET_URL"
    echo HIVE_TAIKO_PRIVATE_KEY: "$HIVE_TAIKO_PRIVATE_KEY"

    export MAINNET_URL="$HIVE_TAIKO_MAINNET_URL"
    export PRIVATE_KEY="$HIVE_TAIKO_PRIVATE_KEY"

    FLAGS="--network mainnet"
    FLAGS="$FLAGS --dao-vault $HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
    FLAGS="$FLAGS --team-vault $HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
    FLAGS="$FLAGS --l2-genesis-block-hash ${l2_genesis_hash}"
    FLAGS="$FLAGS --l2-chain-id ${l2_network_id}"
    FLAGS="$FLAGS --taiko-l2 ${l2_taiko_addr}"
    FLAGS="$FLAGS --confirmations 1"
    echo "Deploy L1 rollup contacts with flags $FLAGS"
    LOG_LEVEL=debug npx hardhat deploy_L1 $FLAGS

    docker cp deployments/mainnet_L1.json "${l1_container_name}:/mainnet_L1.json"

    cd -
    echo "Success to deploy contact on ${containerID}"
}

build_l1_image() {
    docker commit -m "$(whoami)" -m "taiko-l1-image" "${containerID}" taiko-l1:local
    docker container rm -f ${l1_container_name}
    echo "Success to build taiko-l1 image"
}

start_l1_container
deploy_l1_protocol
build_l1_image
