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
	JWTSecret       string
}

var DefaultConfig = &Config{
	L1ChainID:       big.NewInt(1336),
	L2ChainID:       params.TaikoAlpha1NetworkID,
	L1NetworkID:     31336,
	L2NetworkID:     params.TaikoAlpha1NetworkID.Uint64(),
	L1MineInterval:  1,
	TaikoClientTag:  "latest",
	TaikoGethTag:    "taiko",
	ProposeInterval: 30 * time.Second,
	JWTSecret:       "c49690b5a9bc72c7b451b48c5fee2b542e66559d840a133d090769abc56e39e7",
}
