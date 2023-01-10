#!/bin/sh

jq -r 'with_entries(select(.value.contractName=="'$1'"))|keys|.[0]' /deployments/mainnet.json
