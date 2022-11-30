package taiko

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/params"
)

// TODO(alex): split these config
type Config struct {
	L1ChainID       *big.Int
	L2ChainID       *big.Int
	L1NetworkID     uint64
	L2NetworkID     uint64
	L1MineInterval  uint64
	TaikoClientTag  string
	TaikoGethTag    string
	ProposeInterval time.Duration
}

var DefaultConfig = &Config{
	L1ChainID:       big.NewInt(1336),
	L2ChainID:       params.TaikoAlpha1NetworkID,
	L1NetworkID:     31336,
	L2NetworkID:     params.TaikoAlpha1NetworkID.Uint64(),
	L1MineInterval:  0,
	TaikoClientTag:  "latest",
	TaikoGethTag:    "taiko",
	ProposeInterval: time.Second,
}
