package taiko

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/hive/hivesim"
)

type L1DeployConfig struct {
	Deployer         *Account
	RollupAddress    common.Address
	BridgeAddress    common.Address
	VaultAddress     common.Address
	TestERC20Address common.Address
}

type L2DeployConfig struct {
	Throwawayer           *Account // l2 driver account for throwaway invalid block
	SuggestedFeeRecipient *Account // suggested fee recipient account
	Prover                *Account // l1 prover account for prove zk proof
	Proposer              *Account // l1 proposer account for propose l1 txList

	RollupAddress    common.Address
	BridgeAddress    common.Address
	VaultAddress     common.Address
	GenesisBlockHash common.Hash
	TestERC20Address common.Address
}
type DeployConfig struct {
	L1 *L1DeployConfig
	L2 *L2DeployConfig
}

func DefaultDeployments(t *hivesim.T) *DeployConfig {
	c := &DeployConfig{
		L1: &L1DeployConfig{
			Deployer:         NewAccount(t, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			RollupAddress:    common.HexToAddress("0x232e1128a21BBfFbC8d6BefaCb10137F37A653a0"),
			BridgeAddress:    common.HexToAddress("0xAE4C9bD0f7AE5398Df05043079596E2BF0079CE9"),
			VaultAddress:     common.HexToAddress("0x5E506e2E0EaD3Ff9d93859A5879cAA02582f77c3"),
			TestERC20Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		},
		L2: &L2DeployConfig{
			SuggestedFeeRecipient: NewAccount(t, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			Proposer:              NewAccount(t, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			Prover:                NewAccount(t, "6bff9a8ffd7f94f43f4f5f642be8a3f32a94c1f316d90862884b2e276293b6ee"),
			RollupAddress:         common.HexToAddress("0x0000777700000000000000000000000000000001"),
			BridgeAddress:         common.HexToAddress("0x0000777700000000000000000000000000000004"),
			VaultAddress:          common.HexToAddress("0x0000777700000000000000000000000000000002"),
			GenesisBlockHash:      common.HexToHash("0xe76a69b167bb118c79ed3be2dd2073b300acd4861e31a603c7677186b42372a5"),
			TestERC20Address:      common.HexToAddress("0x0000777700000000000000000000000000000005"),
			Throwawayer:           NewAccount(t, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
		},
	}
	return c
}
