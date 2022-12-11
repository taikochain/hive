package taiko

import (
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/hive/hivesim"
)

func WithNoCheck() NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			"HIVE_CHECK_LIVE_PORT": "0",
		})
		return n
	}
}
func WithELNodeType(typ string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envNodeType: typ,
		})
		return n
	}
}

func WithNetworkID(id uint64) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envNetworkID: strconv.FormatUint(id, 10),
		})
		return n
	}
}

func WithLogLevel(level string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envLogLevel: level,
		})
		return n
	}
}

func WithBootNode(nodes string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envBootNode: nodes,
		})
		return n
	}
}

func WithCliquePeriod(seconds uint64) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envCliquePeriod: strconv.FormatUint(seconds, 10),
		})
		return n
	}
}

func WithL1ChainID(chainID *big.Int) NodeOption {
	return func(tn *Node) *Node {
		tn.opts = append(tn.opts, hivesim.Params{
			envTaikoL1ChainID: chainID.String(),
		})
		return tn
	}

}

func WithRole(role string) NodeOption {
	return func(n *Node) *Node {
		n.role = role
		n.opts = append(n.opts, hivesim.Params{envTaikoRole: role})
		return n
	}
}

func WithL1Endpoint(url string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL1RPCEndpoint: url,
		})
		return n
	}
}

func WithL2Endpoint(url string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL2RPCEndpoint: url,
		})
		return n
	}
}

func WithL2EngineEndpoint(url string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL2EngineEndpoint: url,
		})
		return n
	}
}

func WithL1ContractAddress(addr common.Address) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL1RollupAddress: addr.Hex(),
		})
		return n
	}
}

func WithL2ContractAddress(addr common.Address) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL2RollupAddress: addr.Hex(),
		})
		return n
	}
}

func WithThrowawayBlockBuilderPrivateKey(key string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoThrowawayBlockBuilderPrivateKey: key,
		})
		return n
	}
}

func WithEnableL2P2P() NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			evnTaikoEnableL2P2P: "true",
		})
		return n
	}
}

func WithJWTSecret(key string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoJWTSecret: key,
		})
		return n
	}
}

func WithProposerPrivateKey(key string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoProposerPrivateKey: key,
		})
		return n
	}
}

func WithSuggestedFeeRecipient(add common.Address) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoSuggestedFeeRecipient: add.Hex(),
		})
		return n
	}
}

func WithProposeInterval(t time.Duration) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoProposeInterval: t.String(),
		})
		return n
	}
}

func WithProduceInvalidBlocksInterval(seconds uint64) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(seconds, 10),
		})
		return n
	}
}

func WithProverPrivateKey(key string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoProverPrivateKey: key,
		})
		return n
	}
}

func WithPrivateKey(key string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoPrivateKey: key,
		})
		return n
	}
}

func WithL1DeployerAddress(add common.Address) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL1DeployerAddress: add.Hex(),
		})
		return n
	}
}

func WithL2GenesisBlockHash(hash common.Hash) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL2GenesisBlockHash: hash.Hex(),
		})
		return n
	}
}

func WithMainnetUrl(url string) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoMainnetUrl: url,
		})
		return n
	}
}

func WithL2ChainID(chainID *big.Int) NodeOption {
	return func(n *Node) *Node {
		n.opts = append(n.opts, hivesim.Params{
			envTaikoL2ChainID: chainID.String(),
		})
		return n
	}
}
