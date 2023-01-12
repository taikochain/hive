package taiko

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/taikoxyz/taiko-client/bindings"
)

type taikoConfig struct {
	L1NetworkID    uint64 `json:"l1_network_id"`
	L1CliquePeriod uint64 `json:"l1_clique_period"`
	DeployPrivKey  string `json:"deploy_private_key"`
	ProverPrivKey  string `json:"prover_private_key"`
	JWTSecret      string `json:"jwt_secret"`
}

var (
	l1NetworkID      = uint64(31336)
	cliquePeriod     = uint64(0)
	deployAccount, _ = NewAccount("2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200")
	proverAccount, _ = NewAccount("6bff9a8ffd7f94f43f4f5f642be8a3f32a94c1f316d90862884b2e276293b6ee")
	jwtSecret        = "c49690b5a9bc72c7b451b48c5fee2b542e66559d840a133d090769abc56e39e7"

	throwawayAccount, _ = NewAccount(bindings.GoldenTouchPrivKey[2:])
)

func DefaultConfig() *Config {
	return &Config{
		L1: &L1Config{
			ChainID:      big.NewInt(int64(l1NetworkID)),
			NetworkID:    l1NetworkID,
			Deployer:     deployAccount,
			CliquePeriod: cliquePeriod,
		},
		L2: &L2Config{
			ChainID:   params.TaikoAlpha1NetworkID,
			NetworkID: params.TaikoAlpha1NetworkID.Uint64(),
			JWTSecret: jwtSecret,

			Proposer:              deployAccount,
			ProposeInterval:       time.Second,
			SuggestedFeeRecipient: deployAccount,

			Prover: proverAccount,

			Throwawayer: throwawayAccount,
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
	CliquePeriod uint64
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
