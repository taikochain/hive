#!/bin/sh

# Startup script to initialize and boot a go-ethereum instance.
#
# This script assumes the following files:
#  - `geth` binary is located in the filesystem root
#  - `genesis.json` file is located in the filesystem root (mandatory)
#  - `chain.rlp` file is located in the filesystem root (optional)
#  - `blocks` folder is located in the filesystem root (optional)
#  - `keys` folder is located in the filesystem root (optional)
#
# This script assumes the following environment variables:
#
#  - HIVE_BOOTNODE                enode URL of the remote bootstrap node
#  - HIVE_NETWORK_ID              network ID number to use for the eth protocol
#  - HIVE_TESTNET                 whether testnet nonces (2^20) are needed
#  - HIVE_NODETYPE                sync and pruning selector (archive, full, light)
#
# Forks:
#
#  - HIVE_FORK_HOMESTEAD          block number of the homestead hard-fork transition
#  - HIVE_FORK_DAO_BLOCK          block number of the DAO hard-fork transition
#  - HIVE_FORK_DAO_VOTE           whether the node support (or opposes) the DAO fork
#  - HIVE_FORK_TANGERINE          block number of Tangerine Whistle transition
#  - HIVE_FORK_SPURIOUS           block number of Spurious Dragon transition
#  - HIVE_FORK_BYZANTIUM          block number for Byzantium transition
#  - HIVE_FORK_CONSTANTINOPLE     block number for Constantinople transition
#  - HIVE_FORK_PETERSBURG         block number for ConstantinopleFix/PetersBurg transition
#  - HIVE_FORK_ISTANBUL           block number for Istanbul transition
#  - HIVE_FORK_MUIRGLACIER        block number for Muir Glacier transition
#  - HIVE_FORK_BERLIN             block number for Berlin transition
#  - HIVE_FORK_LONDON             block number for London
#
# Clique PoA:
#
#  - HIVE_CLIQUE_PERIOD           enables clique support. value is block time in seconds.
#  - HIVE_CLIQUE_PRIVATEKEY       private key for clique mining
#
# Other:
#
#  - HIVE_MINER                   enable mining. value is coinbase address.
#  - HIVE_MINER_EXTRA             extra-data field to set for newly minted blocks
#  - HIVE_SKIP_POW                if set, skip PoW verification during block import
#  - HIVE_LOGLEVEL                client loglevel (0-5)
#  - HIVE_GRAPHQL_ENABLED         enables graphql on port 8545
#  - HIVE_LES_SERVER              set to '1' to enable LES server

# Taiko environment variables
#
#  - HIVE_TAIKO_L1_RPC_ENDPOINT                      rpc endpoint of the l1 node
#  - HIVE_TAIKO_L2_RPC_ENDPOINT                      rpc endpoint of the l2 node
#  - HIVE_TAIKO_L2_ENGINE_ENDPOINT                   engine endpoint of the l2 node
#  - HIVE_TAIKO_L1_ROLLUP_ADDRESS                    rollup address of the l1 node
#  - HIVE_TAIKO_L2_ROLLUP_ADDRESS                    rollup address of the l2 node
#  - HIVE_TAIKO_PROPOSER_PRIVATE_KEY                 private key of the proposer
#  - HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT              suggested fee recipient
#  - HIVE_TAIKO_PROPOSE_INTERVAL                     propose interval
#  - HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY  private key of the throwaway block builder
#  - HIVE_TAIKO_PROVER_PRIVATE_KEY                   private key of the prover

set -e

geth \
  --nodiscover \
  --gcmode archive \
  --networkid "$HIVE_NETWORK_ID" \
  --http \
  --http.addr 0.0.0.0 \
  --http.vhosts l1_geth \
  --http.vhosts=* \
  --http.api debug,eth,net,web3,txpool,miner \
  --ws \
  --ws.addr 0.0.0.0 \
  --ws.origins '*' \
  --ws.api debug,eth,net,web3,txpool,miner \
  --allow-insecure-unlock \
  --password /dev/null \
  --unlock 0xdf08f82de32b8d460adbe8d72043e3a7e25a3b39 \
  --verbosity 2 \
  --mine \
  --miner.etherbase 0xdf08f82de32b8d460adbe8d72043e3a7e25a3b39
