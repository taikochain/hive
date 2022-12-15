#!/bin/sh

# workspace dir
cd /taiko-mono/packages/protocol

echo HIVE_TAIKO_MAINNET_URL: "$HIVE_TAIKO_MAINNET_URL"
echo HIVE_TAIKO_PRIVATE_KEY: "$HIVE_TAIKO_PRIVATE_KEY"

export MAINNET_URL="$HIVE_TAIKO_MAINNET_URL"
export PRIVATE_KEY="$HIVE_TAIKO_PRIVATE_KEY"

FLAGS="--network mainnet"
FLAGS="$FLAGS --dao-vault $HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
FLAGS="$FLAGS --team-vault $HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
FLAGS="$FLAGS --l2-genesis-block-hash $HIVE_TAIKO_L2_GENESIS_BLOCK_HASH"
FLAGS="$FLAGS --l2-chain-id $HIVE_TAIKO_L2_CHAIN_ID"
FLAGS="$FLAGS --taiko-l2 $HIVE_TAIKO_L2_ROLLUP_ADDRESS"
FLAGS="$FLAGS --confirmations 1"

echo "Deploy L1 rollup contacts with flags $FLAGS"

K_MAX_NUM_BLOCKS=100 K_INITIAL_UNCLE_DELAY=1 npx hardhat preprocess && npx hardhat compile && npx hardhat deploy_L1 $FLAGS
