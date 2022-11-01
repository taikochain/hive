#!/bin/bash

set -eou pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"

echo ""
echo "TAIKO_MONO_DIR: ${TAIKO_MONO_DIR}"
echo "TAIKO_GETH_DIR: ${TAIKO_GETH_DIR}"
echo ""

cd ${TAIKO_GETH_DIR} &&
  make all &&
  cd -

cd ${TAIKO_MONO_DIR}/packages/protocol &&
  yarn clean &&
  yarn compile &&
  cd -

ABIGEN_BIN=$TAIKO_GETH_DIR/build/bin/abigen

echo ""
echo "Start generating go contract bindings..."
echo ""

cat ${TAIKO_MONO_DIR}/packages/protocol/artifacts/contracts/test/TestERC20.sol/TestERC20.json |
  jq .abi |
  ${ABIGEN_BIN} --abi - --type TestERC20 --pkg bindings --out $DIR/../taiko/bindings/test_erc20.go

git -C ${TAIKO_MONO_DIR} log --format="%H" -n 1 >./bindings/.githead

echo "🍻 Go contract bindings generated!"
