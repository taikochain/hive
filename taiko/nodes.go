package taiko

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
)

// These ports are exposed by the docker containers, and accessible via the docker network that the hive test runs in.
// These are container-ports: they are not exposed to the host,
// and so multiple containers can use the same port.
// Some eth1 client definitions hardcode them, others make them configurable, these should not be changed.

const (
	httpRPCPort = 8545
	wsRPCPort   = 8546
	enginePort  = 8551
)

type Node struct {
	*hivesim.Client
	role      string
	opts      []hivesim.StartOption
	TaikoAddr common.Address
}

type ELNode Node

type NodeOption func(*Node) *Node

func NewNode(t *hivesim.T, c *hivesim.ClientDefinition, opts ...NodeOption) *Node {
	n := new(Node)
	for _, o := range opts {
		o(n)
	}
	n.Client = t.StartClient(c.Name, n.opts...)
	return n
}

func (e *ELNode) HttpRpcEndpoint() string {
	return fmt.Sprintf("http://%v:%d", e.IP, httpRPCPort)
}

func (e *ELNode) EngineEndpoint() string {
	return fmt.Sprintf("http://%v:%d", e.IP, enginePort)
}

func (e *ELNode) WsRpcEndpoint() string {
	// carried over from older merge net ws connection problems, idk why clients are different
	switch e.Type {
	case "besu":
		return fmt.Sprintf("ws://%v:%d/ws", e.IP, wsRPCPort)
	case "nethermind":
		return fmt.Sprintf("http://%v:%d/ws", e.IP, wsRPCPort) // upgrade
	default:
		return fmt.Sprintf("ws://%v:%d", e.IP, wsRPCPort)
	}
}

func (e *ELNode) EthClient(t *hivesim.T) *ethclient.Client {
	cli, err := ethclient.Dial(e.WsRpcEndpoint())
	require.NoError(t, err)
	return cli
}

func (e *ELNode) L1TaikoClient(t *hivesim.T) *bindings.TaikoL1Client {
	c := e.EthClient(t)
	cli, err := bindings.NewTaikoL1Client(e.TaikoAddr, c)
	require.NoError(t, err)
	return cli
}

func (e *ELNode) L2TaikoClient(t *hivesim.T) *bindings.V1TaikoL2Client {
	c := e.EthClient(t)
	cli, err := bindings.NewV1TaikoL2Client(e.TaikoAddr, c)
	require.NoError(t, err)
	return cli
}

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
