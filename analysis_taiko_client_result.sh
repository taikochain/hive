#!/bin/bash

log_dir=$1

for log in "${log_dir}"/*-*.json; do
    if [ ! -f "${log}" ]; then
        echo "log not exists, may be caused by timeout"
        break
    fi
    log_name=$(basename "${log}")
    failed=$(jq '.testCases | map_values(.summaryResult.pass) | with_entries(select(.value==false)) | length' "${log}")
    cp -r "${log_dir}"/* workspace/logs/
    echo "failed: ${failed}, detail: http://hive.a1.taiko.xyz/?page=v-pills-results-tab&suite=${log_name}"
done

rm -r "${log_dir}"
