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

func DefaultConfig() *Config {
	return &Config{
		L1: &L1Config{
			ChainID:   big.NewInt(1336),
			NetworkID: 31336,

			Deployer:     deployAccount,
			MineInterval: 0,
		},
		L2: &L2Config{
			ChainID:   params.TaikoAlpha1NetworkID,
			NetworkID: params.TaikoAlpha1NetworkID.Uint64(),

			SuggestedFeeRecipient: deployAccount,
			Proposer:              deployAccount,
			Prover:                proverAccount,

			Throwawayer: deployAccount,

			ProposeInterval: time.Second,
			JWTSecret:       "c49690b5a9bc72c7b451b48c5fee2b542e66559d840a133d090769abc56e39e7",
		},
	}
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
	ChainID      *big.Int
	NetworkID    uint64
	Deployer     *Account
	MineInterval uint64
}

type L2Config struct {
	ChainID   *big.Int
	NetworkID uint64

	Throwawayer           *Account // L2 driver account for throwaway invalid block
	SuggestedFeeRecipient *Account // suggested fee recipient account
	Prover                *Account // L1 prover account for prove zk proof
	Proposer              *Account // L1 proposer account for propose L1 txList

	ProposeInterval time.Duration
	JWTSecret       string
}

type Config struct {
	L1 *L1Config
	L2 *L2Config
}
