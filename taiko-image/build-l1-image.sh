#!/usr/bin/env bash

set -e

debug=true
project_dir=$(realpath "$(dirname $0)/..")
tmp_dir=${project_dir}/tmp
work_dir=${project_dir}/taiko-image

function delete_container() {
    docker container rm -f $1 >/dev/null
}

function delete_image() {
    docker image rm -f $1 >/dev/null
}

l1_container_name=taiko-l1
l2_container_name=taiko-l2

function get_hive_config() {
    local hive_config="${project_dir}/taiko/config.json"
    l1_network_id=$(jq -r .l1_network_id ${hive_config})
    l2_network_id=$(jq -r .l2_network_id ${hive_config})
    jwt_secret=$(jq -r .jwt_secret ${hive_config})
    l1_clique_period=$(jq -r .l1_clique_period "${hive_config}")
    private_key=$(jq -r .deploy_private_key "${hive_config}")
    l1_deploy_address=$(jq -r .deploy_address "${hive_config}")
}

function wait_l2_up() {
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
    local image_name="taiko-geth:tmp"
    delete_image ${image_name}
    docker build -t ${image_name} "${project_dir}/clients/taiko-geth" >/dev/null
    delete_container ${l2_container_name}
    docker run \
        -d \
        -e HIVE_NETWORK_ID="${l2_network_id}" \
        -e HIVE_TAIKO_JWT_SECRET="${jwt_secret}" \
        -p 28545:8545 \
        --name ${l2_container_name} \
        ${image_name} >/dev/null
    wait_l2_up
    l2_genesis_hash=$(
        curl \
            --silent \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","id":0,"method":"eth_getBlockByNumber","params":["0x0", false]}' \
            localhost:28545 | jq -r .result.hash
    )
    delete_image ${image_name}
}

function get_l2_taiko_Addr() {
    delete_container ${l2_container_name}
    docker run \
        -d \
        --name ${l2_container_name} \
        gcr.io/evmchain/taiko-geth:taiko >/dev/null

    docker cp ${l2_container_name}:/deployments/mainnet.json mainnet.json
    l2_taiko_addr=$(jq -r 'with_entries(select(.value.contractName=="TaikoL2"))|keys|.[0]' mainnet.json)
}

function start_l1_container() {
    echo "Start L1 Container"
    local image_name="taiko-l1:tmp"
    delete_image ${image_name}
    docker build -t ${image_name} "${work_dir}/l1" >/dev/null
    delete_container ${l1_container_name}
    build_container=$(docker run -d \
        --name ${l1_container_name} \
        -e HIVE_TAIKO_L1_CHAIN_ID="${l1_network_id}" \
        -p 18545:8545 ${image_name})
}

mono_dir="${tmp_dir}/taiko-mono"
protocol_dir="${mono_dir}/packages/protocol"

change_protocol() {
    # change some protocol config for test
    local origin="${protocol_dir}/contracts/libs/LibSharedConfig.sol"
    local changed="${work_dir}/LibSharedConfig.sol"
    sed -f "${work_dir}/LibSharedConfig.sed" "${origin}" >"${changed}"
    mv "${changed}" "${origin}"
    # change prove method for test
    cp "${work_dir}/LibZKP.sol" "${protocol_dir}/contracts/libs/LibZKP.sol"
    # Make genesis.json consistent with hive test configuration
    local origin="${work_dir}/genesis.json"
    local changed="${work_dir}/tmp.json"
    jq ".config.chainId=${l1_network_id}" "${origin}" | jq ".config.clique.period=${l1_clique_period}" >"${changed}" && mv "${changed}" "${origin}"
}

download_protocol() {
    if [[ "${debug}" == "true" ]]; then
        return
    fi

    rm -fr "${mono_dir}"
    git clone --depth=1 https://github.com/taikoxyz/taiko-mono.git ${mono_dir}

    change_protocol

    if [ ! -f "${protocol_dir}/bin/solc" ]; then
        ${protocol_dir}/scripts/download_solc.sh
    fi

    cd ${protocol_dir} && pnpm install && K_CHAIN_ID=${l2_network_id} pnpm compile && cd -
}

deploy_protocol() {
    download_protocol

    echo "Start deploying contact on ${build_container}"

    HIVE_TAIKO_MAINNET_URL="http://127.0.0.1:18545"

    get_l2_genesis_hash
    get_l2_taiko_Addr

    echo HIVE_TAIKO_MAINNET_URL: "$HIVE_TAIKO_MAINNET_URL"
    echo private_key: "${private_key}"

    export MAINNET_URL="$HIVE_TAIKO_MAINNET_URL"
    export PRIVATE_KEY="${private_key}"
    local network="l1_test"
    FLAGS="--network ${network}"
    FLAGS="$FLAGS --dao-vault $l1_deploy_address"
    FLAGS="$FLAGS --team-vault $l1_deploy_address"
    FLAGS="$FLAGS --l2-genesis-block-hash ${l2_genesis_hash}"
    FLAGS="$FLAGS --l2-chain-id ${l2_network_id}"
    FLAGS="$FLAGS --taiko-l2 ${l2_taiko_addr}"
    FLAGS="$FLAGS --confirmations 1"

    echo "Deploy L1 rollup contacts with flags $FLAGS"
    cd ${protocol_dir} && LOG_LEVEL=debug npx hardhat deploy_L1 $FLAGS && cd -

    docker cp "${protocol_dir}/deployments/${network}_L1.json" "${l1_container_name}:/mainnet.json"

    echo "Success to deploy contact on ${build_container}"
}

build_l1_image() {
    docker commit -m "$(whoami)" -m "taiko-l1-image" "${build_container}" taiko-l1:local >/dev/null
    delete_container ${l1_container_name}
    echo "Success to build taiko-l1 image"
}

get_hive_config
start_l1_container
deploy_protocol
build_l1_image
