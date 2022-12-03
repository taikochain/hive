package taiko

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/params"
)

type L1Config struct {
	ChainID      *big.Int
	NetworkID    uint64
	MineInterval uint64
}

type L2Config struct {
	ChainID                      *big.Int
	NetworkID                    uint64
	ProduceInvalidBlocksInterval uint64
	ProposeInterval              time.Duration
	JWTSecret                    string
}

type RollupConfig struct {
	L1 *L1Config
	L2 *L2Config
}

var DefaultRollupConfig = &RollupConfig{
	L1: &L1Config{
		ChainID:      big.NewInt(1336),
		NetworkID:    31336,
		MineInterval: 1,
	},
	L2: &L2Config{
		ChainID:         params.TaikoAlpha1NetworkID,
		NetworkID:       params.TaikoAlpha1NetworkID.Uint64(),
		ProposeInterval: 30 * time.Second,
		JWTSecret:       "c49690b5a9bc72c7b451b48c5fee2b542e66559d840a133d090769abc56e39e7",
	},
}

type DevnetConfig struct {
	L1EngineCnt uint64
	L2EngineCnt uint64
	DriverCnt   uint64
	ProposerCnt uint64
	ProverCnt   uint64
}
