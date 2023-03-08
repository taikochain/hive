#!/usr/bin/env bash

set -e

debug=false
project_dir=$(realpath "$(dirname "$0")/..")
tmp_dir=${project_dir}/tmp
work_dir=${project_dir}/taiko-image

client_branch="main"

function choose_client_branch() {
    if [[ "${REPO}" == "taikoxyz/taiko-client" ]]; then
        if [[ -n "${HEAD_REF}" ]]; then
            client_branch=${HEAD_REF}
        elif [[ -n "${REF_NAME}" ]]; then
            client_branch=${REF_NAME}
        fi
    fi
    echo "Current branch:" "${client_branch}"
}

client_dir="${tmp_dir}/taiko-client"

function download_client_repo() {
    if [[ "${debug}" == "true" ]]; then
        print "In debug mode, do not download taiko-client repo"
        return
    fi
    rm -fr "${client_dir}"
    git clone https://github.com/taikoxyz/taiko-client.git "${client_dir}"
}

function build_client_image() {
    cd "${client_dir}"
    git checkout "${client_branch}"
    docker build -t taiko-client:local .
    cd "${work_dir}"
    echo "Success to build taiko-client image"
}

choose_client_branch
download_client_repo
build_client_image
