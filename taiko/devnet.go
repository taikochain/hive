package taiko

import (
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
)

// Devnet is a taiko network with all necessary components, e.g. L1, L2, driver, proposer, prover etc.
type Devnet struct {
	sync.Mutex
	// nodes
	L1Engines []*ELNode
	L2Engines []*ELNode
	drivers   []*Node
	proposers []*Node
	provers   []*Node

	L2Genesis *core.Genesis
}

type DevOption func(*Devnet) *Devnet

func NewDevnet(t *hivesim.T, c *Config, opts ...DevOption) *Devnet {
	d := &Devnet{}
	d.L2Genesis = core.TaikoGenesisBlock(c.L2.NetworkID)
	for _, o := range opts {
		o(d)
	}
	return d
}

func WithL1Node(l1 *ELNode) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.L1Engines = append(d.L1Engines, l1)
		return d
	}
}

func WithL2Node(l2 *ELNode) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.L2Engines = append(d.L2Engines, l2)
		return d
	}
}

func WithDriverNode(n *Node) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.drivers = append(d.drivers, n)
		return d
	}
}

func WithProposerNode(n *Node) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.proposers = append(d.proposers, n)
		return d
	}
}

func WithProverNode(n *Node) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.provers = append(d.provers, n)
		return d
	}
}

// NewL1ELNode starts a eth1 image and add it to the network
func NewL1ELNode(t *hivesim.T, env *TestEnv, l2 *ELNode) *ELNode {
	opts := []NodeOption{
		WithRole("L1Engine"),
		WithL1ChainID(env.Conf.L1.ChainID),
		WithNetworkID(env.Conf.L1.NetworkID),
		WithCliquePeriod(env.Conf.L1.MineInterval),
	}
	n := NewNode(t, env.Clients.L1[0], opts...)
	n.TaikoAddr = env.Conf.L1.RollupAddress
	elNode := (*ELNode)(n)
	WaitELNodesUp(env.Context, t, elNode, 10*time.Second)
	return elNode
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.L1Engines) {
		return nil
	}
	return d.L1Engines[idx]
}

func NewL2ELNode(t *hivesim.T, env *TestEnv, bootNodes string) *ELNode {
	opts := []NodeOption{
		WithRole("L2Engine"),
		WithJWTSecret(env.Conf.L2.JWTSecret),
		WithELNodeType("full"),
		WithNetworkID(env.Conf.L2.NetworkID),
		WithLogLevel("4"),
	}
	if len(bootNodes) > 0 {
		opts = append(opts, WithBootNode(bootNodes))
	}
	n := NewNode(t, env.Clients.L2[0], opts...)
	n.TaikoAddr = env.Conf.L2.RollupAddress
	elNode := (*ELNode)(n)
	WaitELNodesUp(env.Context, t, elNode, 10*time.Second)
	return elNode
}

func (d *Devnet) GetBootNodes(t *hivesim.T) string {
	d.Lock()
	defer d.Unlock()
	urls := make([]string, 0)
	for i, n := range d.L2Engines {
		enodeURL, err := n.EnodeURL()
		if err != nil {
			t.Fatalf("failed to get enode url of the %d taiko geth node, error: %v", i, err)
		}
		urls = append(urls, enodeURL)
	}
	return strings.Join(urls, ",")
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.L2Engines) {
		return nil
	}
	return d.L2Engines[idx]
}

func NewDriverNode(t *hivesim.T, env *TestEnv, l1, l2 *ELNode, enableP2P bool) *Node {
	opts := []NodeOption{
		WithRole("driver"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL2EngineEndpoint(l2.EngineEndpoint()),
		WithL1ContractAddress(env.Conf.L1.RollupAddress),
		WithL2ContractAddress(env.Conf.L2.RollupAddress),
		WithThrowawayBlockBuilderPrivateKey(env.Conf.L2.Throwawayer.PrivateKeyHex),
		WithJWTSecret(env.Conf.L2.JWTSecret),
	}
	if enableP2P {
		opts = append(opts, WithEnableL2P2P())
	}
	n := NewNode(t, env.Clients.Driver[0], opts...)
	return n
}

func NewProposerNode(t *hivesim.T, env *TestEnv, l1, l2 *ELNode) *Node {
	opts := []NodeOption{
		WithRole("proposer"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(env.Conf.L1.RollupAddress),
		WithL2ContractAddress(env.Conf.L2.RollupAddress),
		WithProposerPrivateKey(env.Conf.L2.Proposer.PrivateKeyHex),
		WithSuggestedFeeRecipient(env.Conf.L2.SuggestedFeeRecipient.Address),
		WithProposeInterval(env.Conf.L2.ProposeInterval),
	}

	if env.Conf.L2.ProduceInvalidBlocksInterval != 0 {
		opts = append(opts, WithProduceInvalidBlocksInterval(env.Conf.L2.ProduceInvalidBlocksInterval))
	}
	return NewNode(t, env.Clients.Proposer[0], opts...)
}

func NewProverNode(t *hivesim.T, env *TestEnv, l1, l2 *ELNode) *Node {
	addWhitelist(t, env, l1.EthClient(t))
	opts := []NodeOption{
		WithRole("prover"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(env.Conf.L1.RollupAddress),
		WithL2ContractAddress(env.Conf.L2.RollupAddress),
		WithProverPrivateKey(env.Conf.L2.Prover.PrivateKeyHex),
	}
	return NewNode(t, env.Clients.Prover[0], opts...)
}

func addWhitelist(t *hivesim.T, env *TestEnv, cli *ethclient.Client) {
	taikoL1, err := bindings.NewTaikoL1Client(env.Conf.L1.RollupAddress, cli)
	require.NoError(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(env.Conf.L1.Deployer.PrivateKey, env.Conf.L1.ChainID)
	require.NoError(t, err)

	opts.GasTipCap = big.NewInt(1500000000)
	tx, err := taikoL1.WhitelistProver(opts, env.Conf.L2.Prover.Address, true)
	require.NoError(t, err)
	receipt, err := rpc.WaitReceipt(env.Context, cli, tx)
	require.NoError(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatal("Failed to commit transactions list", "txHash", receipt.TxHash)
	}
	t.Log("Add prover to whitelist finished", "height", receipt.BlockNumber)
}

// deployL1Contracts runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func deployL1Contracts(t *hivesim.T, env *TestEnv, l1Node, l2 *ELNode) {
	l2GenesisHash := GetBlockHashByNumber(env.Context, t, l2.EthClient(t), common.Big0, false)
	opts := []NodeOption{
		WithNoCheck(),
		WithPrivateKey(env.Conf.L1.Deployer.PrivateKeyHex),
		WithL1DeployerAddress(env.Conf.L1.Deployer.Address),
		WithL2GenesisBlockHash(l2GenesisHash),
		WithL2ContractAddress(env.Conf.L2.RollupAddress),
		WithMainnetUrl(l1Node.HttpRpcEndpoint()),
		WithL2ChainID(env.Conf.L2.ChainID),
	}
	n := NewNode(t, env.Clients.Contract, opts...)
	result, err := n.Exec("deploy.sh")
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("failed to deploy contract on engine node %s, error: %v, result: %v",
			l1Node.Container, err, result)
	}
	t.Logf("Deploy contracts on %s %s(%s)", l1Node.Type, l1Node.Container, l1Node.IP)
}
