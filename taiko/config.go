package taiko

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var (
	deployAccount, _ = NewAccount("2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200")
	proverAccount, _ = NewAccount("6bff9a8ffd7f94f43f4f5f642be8a3f32a94c1f316d90862884b2e276293b6ee")
)

var DefaultConfig = &Config{
	L1: &L1Config{
		ChainID:   big.NewInt(1336),
		NetworkID: 31336,

		Deployer:         deployAccount,
		RollupAddress:    common.HexToAddress("0x232e1128a21BBfFbC8d6BefaCb10137F37A653a0"),
		BridgeAddress:    common.HexToAddress("0xAE4C9bD0f7AE5398Df05043079596E2BF0079CE9"),
		VaultAddress:     common.HexToAddress("0x5E506e2E0EaD3Ff9d93859A5879cAA02582f77c3"),
		TestERC20Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),

		MineInterval: 1,
	},
	L2: &L2Config{
		ChainID:   params.TaikoAlpha1NetworkID,
		NetworkID: params.TaikoAlpha1NetworkID.Uint64(),

		SuggestedFeeRecipient: deployAccount,
		Proposer:              deployAccount,
		Prover:                proverAccount,
		RollupAddress:         common.HexToAddress("0x0000777700000000000000000000000000000001"),
		BridgeAddress:         common.HexToAddress("0x0000777700000000000000000000000000000004"),
		VaultAddress:          common.HexToAddress("0x0000777700000000000000000000000000000002"),
		GenesisBlockHash:      common.HexToHash("0xe76a69b167bb118c79ed3be2dd2073b300acd4861e31a603c7677186b42372a5"),
		TestERC20Address:      common.HexToAddress("0x0000777700000000000000000000000000000005"),
		Throwawayer:           deployAccount,

		ProposeInterval: time.Second,
		JWTSecret:       "c49690b5a9bc72c7b451b48c5fee2b542e66559d840a133d090769abc56e39e7",
	},
}

type Account struct {
	PrivateKeyHex string
	PrivateKey    *ecdsa.PrivateKey
	Address       common.Address
}

func NewAccount(privKeyHex string) (*Account, error) {
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	return &Account{
		PrivateKeyHex: privKeyHex,
		PrivateKey:    privKey,
		Address:       addr,
	}, nil
}

type L1Config struct {
	ChainID   *big.Int
	NetworkID uint64

	Deployer         *Account
	RollupAddress    common.Address
	BridgeAddress    common.Address
	VaultAddress     common.Address
	TestERC20Address common.Address

	MineInterval uint64
}

type L2Config struct {
	ChainID   *big.Int
	NetworkID uint64

	Throwawayer           *Account // L2 driver account for throwaway invalid block
	SuggestedFeeRecipient *Account // suggested fee recipient account
	Prover                *Account // L1 prover account for prove zk proof
	Proposer              *Account // L1 proposer account for propose L1 txList
	RollupAddress         common.Address
	BridgeAddress         common.Address
	VaultAddress          common.Address
	GenesisBlockHash      common.Hash
	TestERC20Address      common.Address

	ProduceInvalidBlocksInterval uint64
	ProposeInterval              time.Duration
	JWTSecret                    string
}

type Config struct {
	L1 *L1Config
	L2 *L2Config
}
