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
#  - HIVE_TAIKO_NETWORK_ID                           network ID number to use for the taiko protocol
#  - HIVE_TAIKO_BOOTNODE                             enode URL of the remote bootstrap node for l2 node
#  - HIVE_TAIKO_ROLE                                 role of node
#  - HIVE_TAIKO_L1_RPC_ENDPOINT                      rpc endpoint of the l1 node
#  - HIVE_TAIKO_L2_RPC_ENDPOINT                      rpc endpoint of the l2 node
#  - HIVE_TAIKO_L2_ENGINE_ENDPOINT                   engine endpoint of the l2 node
#  - HIVE_TAIKO_L1_ROLLUP_ADDRESS                    rollup address of the l1 node
#  - HIVE_TAIKO_L2_ROLLUP_ADDRESS                    rollup address of the l2 node
#  - HIVE_TAIKO_PROPOSER_PRIVATE_KEY                 private key of the proposer
#  - HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT              suggested fee recipient
#  - HIVE_TAIKO_PROPOSE_INTERVAL                     propose interval
#  - HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY  private key of the throwaway block builder
#  - HIVE_TAIKO_L1_CHAIN_ID                          l1 chain id
#  - HIVE_TAIKO_L1_CLIQUE_PERIOD                     l1 clique period
#  - HIVE_TAIKO_PROVER_PRIVATE_KEY                   private key of the prover
#  - HIVE_TAIKO_JWT_SECRET                           jwt secret used by driver and taiko geth

set -e

echo "start $HIVE_TAIKO_ROLE ..."

case $HIVE_TAIKO_ROLE in
"taiko-driver")
  echo $HIVE_TAIKO_JWT_SECRET >/jwtsecret
  taiko-client driver \
    --l1 "$HIVE_TAIKO_L1_RPC_ENDPOINT" \
    --l2 "$HIVE_TAIKO_L2_RPC_ENDPOINT" \
    --l2.engine "$HIVE_TAIKO_L2_ENGINE_ENDPOINT" \
    --taikoL1 "$HIVE_TAIKO_L1_ROLLUP_ADDRESS" \
    --taikoL2 "$HIVE_TAIKO_L2_ROLLUP_ADDRESS" \
    --l2.throwawayBlockBuilderPrivKey "$HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY" \
    --jwtSecret /jwtsecret \
    --verbosity 4
  ;;
"taiko-prover")
  taiko-client prover \
    --l1 "$HIVE_TAIKO_L1_RPC_ENDPOINT" \
    --l2 "$HIVE_TAIKO_L2_RPC_ENDPOINT" \
    --taikoL1 "$HIVE_TAIKO_L1_ROLLUP_ADDRESS" \
    --taikoL2 "$HIVE_TAIKO_L2_ROLLUP_ADDRESS" \
    --zkevmRpcdEndpoint ws://127.0.0.1:18545 \
    --zkevmRpcdParamsPath 12345 \
    --l1.proverPrivKey "$HIVE_TAIKO_PROVER_PRIVATE_KEY" \
    --verbosity 4 \
    --dummy
  ;;
"taiko-proposer")
  if [ "$HIVE_TAIKO_PRODUCE_INVALID_BLOCKS_INTERVAL" != "" ]; then
    taiko-client proposer \
      --l1 "$HIVE_TAIKO_L1_RPC_ENDPOINT" \
      --l2 "$HIVE_TAIKO_L2_RPC_ENDPOINT" \
      --taikoL1 "$HIVE_TAIKO_L1_ROLLUP_ADDRESS" \
      --taikoL2 "$HIVE_TAIKO_L2_ROLLUP_ADDRESS" \
      --l1.proposerPrivKey "$HIVE_TAIKO_PROPOSER_PRIVATE_KEY" \
      --l2.suggestedFeeRecipient "$HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT" \
      --proposeInterval "$HIVE_TAIKO_PROPOSE_INTERVAL" \
      --verbosity $HIVE_LOGLEVEL \
      --produceInvalidBlocks \
      --produceInvalidBlocksInterval "$HIVE_TAIKO_PRODUCE_INVALID_BLOCKS_INTERVAL"
  else
    taiko-client proposer \
      --l1 "$HIVE_TAIKO_L1_RPC_ENDPOINT" \
      --l2 "$HIVE_TAIKO_L2_RPC_ENDPOINT" \
      --taikoL1 "$HIVE_TAIKO_L1_ROLLUP_ADDRESS" \
      --taikoL2 "$HIVE_TAIKO_L2_ROLLUP_ADDRESS" \
      --l1.proposerPrivKey "$HIVE_TAIKO_PROPOSER_PRIVATE_KEY" \
      --l2.suggestedFeeRecipient "$HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT" \
      --proposeInterval "$HIVE_TAIKO_PROPOSE_INTERVAL" \
      --verbosity $HIVE_LOGLEVEL
  fi
  ;;
esac
