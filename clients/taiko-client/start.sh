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
#  - HIVE_TAIKO_ROLE                                 role of node
#  - HIVE_TAIKO_L1_CHAIN_ID                          l1 chain id
#  - HIVE_TAIKO_JWT_SECRET                           jwt secret used by driver and taiko geth
#  - HIVE_TAIKO_L1_RPC_ENDPOINT                      rpc endpoint of the l1 node
#  - HIVE_TAIKO_L2_RPC_ENDPOINT                      rpc endpoint of the l2 node
#  - HIVE_TAIKO_L1_ROLLUP_ADDRESS                    rollup address of the l1 node
#  - HIVE_TAIKO_L2_ROLLUP_ADDRESS                    rollup address of the l2 node

# Driver
#  - HIVE_TAIKO_L2_ENGINE_ENDPOINT                   engine endpoint of the l2 node
#  - HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY  private key of the throwaway block builder
#  - HIVE_TAIKO_ENABLE_L2_P2P                        enable taiko geth sync blocks by p2p

# Proposer
#  - HIVE_TAIKO_PROPOSER_PRIVATE_KEY                 private key of the proposer
#  - HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT              suggested fee recipient
#  - HIVE_TAIKO_PROPOSE_INTERVAL                     propose interval

# Prover
#  - HIVE_TAIKO_PROVER_PRIVATE_KEY                   private key of the prover

# Deployer
# - HIVE_TAIKO_L1_DEPLOYER_ADDRESS
# - HIVE_TAIKO_L2_GENESIS_BLOCK_HASH
# - HIVE_TAIKO_MAINNET_URL
# - HIVE_TAIKO_PRIVATE_KEY
# - HIVE_TAIKO_L2_CHAIN_ID

set -e

FLAGS="--l1.ws $HIVE_TAIKO_L1_RPC_ENDPOINT --l2.ws $HIVE_TAIKO_L2_RPC_ENDPOINT"
FLAGS="$FLAGS --taikoL1 $HIVE_TAIKO_L1_ROLLUP_ADDRESS  --taikoL2 $HIVE_TAIKO_L2_ROLLUP_ADDRESS"
FLAGS="$FLAGS --verbosity $HIVE_LOGLEVEL"

case $HIVE_TAIKO_ROLE in
"driver")
  echo "$HIVE_TAIKO_JWT_SECRET" >/jwtsecret
  FLAGS="$FLAGS --l2.auth $HIVE_TAIKO_L2_ENGINE_ENDPOINT"
  FLAGS="$FLAGS --l2.throwawayBlockBuilderPrivKey=$HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY"
  FLAGS="$FLAGS --jwtSecret /jwtsecret"
  if [ "$HIVE_TAIKO_ENABLE_L2_P2P" != "" ]; then
    FLAGS="$FLAGS --p2p.syncVerifiedBlocks"
  fi
  ;;
"prover")
  FLAGS="$FLAGS --zkevmRpcdEndpoint=ws://127.0.0.1:18545"
  FLAGS="$FLAGS --zkevmRpcdParamsPath=12345"
  FLAGS="$FLAGS --l1.proverPrivKey=$HIVE_TAIKO_PROVER_PRIVATE_KEY"
  FLAGS="$FLAGS --dummy"
  ;;
"proposer")
  FLAGS="$FLAGS --l1.proposerPrivKey=$HIVE_TAIKO_PROPOSER_PRIVATE_KEY"
  FLAGS="$FLAGS --l2.suggestedFeeRecipient=$HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT"
  FLAGS="$FLAGS --proposeInterval=$HIVE_TAIKO_PROPOSE_INTERVAL"
  if [ "$HIVE_TAIKO_PRODUCE_INVALID_BLOCKS_INTERVAL" != "" ]; then
    FLAGS="$FLAGS --produceInvalidBlocks"
    FLAGS="$FLAGS --produceInvalidBlocksInterval=$HIVE_TAIKO_PRODUCE_INVALID_BLOCKS_INTERVAL"
  fi
  ;;
esac

# Run the go-ethereum implementation with the requested flags.
echo "Running $HIVE_TAIKO_ROLE with flags $FLAGS"
taiko-client $HIVE_TAIKO_ROLE $FLAGS
