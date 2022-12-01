package taiko

import "github.com/ethereum/go-ethereum/common"

// TODO(alex): split two structs
type Deployments struct {
	// rollup contract address
	L1RollupAddress common.Address
	L2RollupAddress common.Address
	// bridge contract address
	L1BridgeAddress common.Address
	L2BridgeAddress common.Address
	// vault contract address
	L1VaultAddress     common.Address
	L2VaultAddress     common.Address
	L2GenesisBlockHash common.Hash
	// TestERC20 contract address
	L1TestERC20Address common.Address
	L2TestERC20Address common.Address
}

var (
	DefaultDeployments = &Deployments{
		L1RollupAddress:    common.HexToAddress("0x232e1128a21BBfFbC8d6BefaCb10137F37A653a0"),
		L1BridgeAddress:    common.HexToAddress("0xAE4C9bD0f7AE5398Df05043079596E2BF0079CE9"),
		L1VaultAddress:     common.HexToAddress("0x5E506e2E0EaD3Ff9d93859A5879cAA02582f77c3"),
		L2RollupAddress:    common.HexToAddress("0x0000777700000000000000000000000000000001"),
		L2BridgeAddress:    common.HexToAddress("0x0000777700000000000000000000000000000004"),
		L2VaultAddress:     common.HexToAddress("0x0000777700000000000000000000000000000002"),
		L2GenesisBlockHash: common.HexToHash("0xe76a69b167bb118c79ed3be2dd2073b300acd4861e31a603c7677186b42372a5"),
		// TODO: update these addresses
		L1TestERC20Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		L2TestERC20Address: common.HexToAddress("0x0000777700000000000000000000000000000005"),
	}
)
