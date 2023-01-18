#!/usr/bin/env bash

set -e

# source ./common.sh
tmp_dir=/mnt/disks/data/tmp

client_branch="main"
if [[ "${REPO}" == "taikoxyz/taiko-client" ]]; then
    if [[ "${REF}" == "refs/pull" ]]; then
        echo "${REF} contains: refs/pull"
    fi
    client_branch=${HEAD_REF}
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
