#!/usr/bin/env bash

set -e

# source ./common.sh
tmp_dir=/mnt/disks/data/tmp

client_branch="main"
if [[ "${REPO}" == "taikoxyz/taiko-client" ]]; then
    if [[ "${HEAD_REF}" != "" ]]; then
        client_branch=${HEAD_REF}
    elif [[ "${REF_NAME}" != "" ]]; then
        client_branch=${REF_NAME}
    fi
fi

build_client_image() {
    client_dir="${tmp_dir}/taiko-client"
    rm -fr "${client_dir}"
    git clone --depth=1 https://github.com/taikoxyz/taiko-client.git "${client_dir}"
    cd "${client_dir}"
    git checkout "${client_branch}"
    docker build -t taiko-client:local .
    cd -
    echo "Success to build taiko-client image"
}

build_client_image
