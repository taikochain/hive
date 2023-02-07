#!/usr/bin/env bash

set -e

debug=false

tmp_dir=/mnt/disks/data/tmp

client_branch="main"
if [[ "${REPO}" == "taikoxyz/taiko-client" ]]; then
    if [[ -n "${HEAD_REF}" ]]; then
        client_branch=${HEAD_REF}
    elif [[ -n "${REF_NAME}" ]]; then
        client_branch=${REF_NAME}
    fi
fi

build_client_image() {
    if [[ "${debug}" == "true" ]]; then
        client_dir=$(
            cd ../taiko-client
            pwd
        )
    else
        client_dir="${tmp_dir}/taiko-client"
        rm -fr "${client_dir}"
        git clone https://github.com/taikoxyz/taiko-client.git "${client_dir}"
    fi
    cd "${client_dir}"
    git checkout "${client_branch}"
    docker build -t taiko-client:local .
    cd -
    echo "Success to build taiko-client image"
}

build_client_image
