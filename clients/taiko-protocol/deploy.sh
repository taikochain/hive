#!/bin/sh

# workspace dir
cd /taiko-mono/packages/protocol

echo "$HIVE_TAIKO_MAINNET_URL"
echo "$HIVE_TAIKO_PRIVATE_KEY"
echo "$HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
echo "$HIVE_TAIKO_L2_GENESIS_BLOCK_HASH"
echo "$HIVE_TAIKO_L2_CHAIN_ID"
echo "$HIVE_TAIKO_L2_ROLLUP_ADDRESS"

export MAINNET_URL="$HIVE_TAIKO_MAINNET_URL"
export PRIVATE_KEY="$HIVE_TAIKO_PRIVATE_KEY"

npx hardhat deploy_L1 \
  --network mainnet \
  --dao-vault "$HIVE_TAIKO_L1_DEPLOYER_ADDRESS" \
  --team-vault "$HIVE_TAIKO_L1_DEPLOYER_ADDRESS" \
  --l2-genesis-block-hash "$HIVE_TAIKO_L2_GENESIS_BLOCK_HASH" \
  --l2-chain-id "$HIVE_TAIKO_L2_CHAIN_ID" \
  --v1-taiko-l2 "$HIVE_TAIKO_L2_ROLLUP_ADDRESS" \
  --confirmations 1
