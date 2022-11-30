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
		L2RollupAddress:    common.HexToAddress("0xb0cA4AEFaDEc43cBd7761105d0dc16b0D8373094"),
		L1BridgeAddress:    common.HexToAddress("0xB12d6112D64B213880Fa53F815aF1F29c91CaCe9"),
		L2BridgeAddress:    common.HexToAddress("0xcBECCC94C76EDb11aAb917cb4eD6D633DA84C176"),
		L1VaultAddress:     common.HexToAddress("0xDA1Ea1362475997419D2055dD43390AEE34c6c37"),
		L2VaultAddress:     common.HexToAddress("0x2B754E21D363B4005eF7e7679F3B2D140B4B00Fc"),
		L2GenesisBlockHash: common.HexToHash("0xe76a69b167bb118c79ed3be2dd2073b300acd4861e31a603c7677186b42372a5"),
		// TODO: update these addresses
		L1TestERC20Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		L2TestERC20Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	}
)
