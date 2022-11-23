package taiko

// Taiko environment variables
//
//  - HIVE_TAIKO_NETWORK_ID                           network ID number to use for the taiko protocol
//  - HIVE_TAIKO_BOOTNODE                             enode URL of the remote bootstrap node for l2 node
//  - HIVE_TAIKO_L1_RPC_ENDPOINT                      rpc endpoint of the l1 node
//  - HIVE_TAIKO_L2_RPC_ENDPOINT                      rpc endpoint of the l2 node
//  - HIVE_TAIKO_L2_ENGINE_ENDPOINT                   engine endpoint of the l2 node
//  - HIVE_TAIKO_L1_ROLLUP_ADDRESS                    rollup address of the l1 node
//  - HIVE_TAIKO_L2_ROLLUP_ADDRESS                    rollup address of the l2 node
//  - HIVE_TAIKO_PROPOSER_PRIVATE_KEY                 private key of the proposer
//  - HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT              suggested fee recipient
//  - HIVE_TAIKO_PROPOSE_INTERVAL                     propose interval
//  - HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY  private key of the throwaway block builder
//  - HIVE_TAIKO_L1_CHAIN_ID                          l1 chain id
//  - HIVE_TAIKO_L1_CLIQUE_PERIOD                     l1 clique period
//  - HIVE_TAIKO_PROVER_PRIVATE_KEY                   private key of the prover

// taiko environment variables constants
const (
	envTaikoNetworkId                       = "HIVE_TAIKO_NETWORK_ID"
	envTaikoBootNode                        = "HIVE_TAIKO_BOOTNODE"
	envTaikoL1RPCEndpoint                   = "HIVE_TAIKO_L1_RPC_ENDPOINT"
	envTaikoL2RPCEndpoint                   = "HIVE_TAIKO_L2_RPC_ENDPOINT"
	envTaikoL2EngineEndpoint                = "HIVE_TAIKO_L2_ENGINE_ENDPOINT"
	envTaikoL1RollupAddress                 = "HIVE_TAIKO_L1_ROLLUP_ADDRESS"
	envTaikoL2RollupAddress                 = "HIVE_TAIKO_L2_ROLLUP_ADDRESS"
	envTaikoProposerPrivateKey              = "HIVE_TAIKO_PROPOSER_PRIVATE_KEY"
	envTaikoSuggestedFeeRecipient           = "HIVE_TAIKO_SUGGESTED_FEE_RECIPIENT"
	envTaikoProposeInterval                 = "HIVE_TAIKO_PROPOSE_INTERVAL"
	envTaikoProduceInvalidBlocksInterval    = "HIVE_TAIKO_PRODUCE_INVALID_BLOCKS_INTERVAL"
	envTaikoThrowawayBlockBuilderPrivateKey = "HIVE_TAIKO_THROWAWAY_BLOCK_BUILDER_PRIVATE_KEY"
	envTaikoL1ChainId                       = "HIVE_TAIKO_L1_CHAIN_ID"
	envTaikoL1CliquePeriod                  = "HIVE_TAIKO_L1_CLIQUE_PERIOD"
	envTaikoProverPrivateKey                = "HIVE_TAIKO_PROVER_PRIVATE_KEY"

	// deployer
	envTaikoL1DeployerAddress  = "HIVE_TAIKO_L1_DEPLOYER_ADDRESS"
	envTaikoL2GenesisBlockHash = "HIVE_TAIKO_L2_GENESIS_BLOCK_HASH"
	envTaikoMainnetUrl         = "HIVE_TAIKO_MAINNET_URL"
	envTaikoPrivateKey         = "HIVE_TAIKO_PRIVATE_KEY"
	envTaikoL2ChainId          = "HIVE_TAIKO_L2_CHAIN_ID"
)
