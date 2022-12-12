package taiko

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
)

type Node struct {
	*hivesim.Client
	role      string
	opts      []hivesim.StartOption
	TaikoAddr common.Address
}

type NodeOption func(*Node) *Node

func NewNode(t *hivesim.T, c *hivesim.ClientDefinition, opts ...NodeOption) *Node {
	n := new(Node)
	for _, o := range opts {
		o(n)
	}
	n.Client = t.StartClient(c.Name, n.opts...)
	return n
}

// NewL1ELNode starts a eth1 image and add it to the network
func NewL1ELNode(t *hivesim.T, env *TestEnv) *ELNode {
	require.NotNil(t, env.Clients.L1)
	opts := []NodeOption{
		WithRole("L1Engine"),
		WithL1ChainID(env.Conf.L1.ChainID),
		WithNetworkID(env.Conf.L1.NetworkID),
		WithCliquePeriod(env.Conf.L1.MineInterval),
	}
	n := NewNode(t, env.Clients.L1, opts...)
	n.TaikoAddr = env.Conf.L1.RollupAddress
	elNode := (*ELNode)(n)
	WaitELNodesUp(env.Context, t, elNode, 10*time.Second)
	return elNode
}

// deployL1Contracts runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func deployL1Contracts(t *hivesim.T, env *TestEnv, l1Node, l2 *ELNode) {
	require.NotNil(t, env.Clients.Contract)
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

func NewL2ELNode(t *hivesim.T, env *TestEnv, bootNodes string) *ELNode {
	require.NotNil(t, env.Clients.L2)
	opts := []NodeOption{
		WithRole("L2Engine"),
		WithJWTSecret(env.Conf.L2.JWTSecret),
		WithELNodeType("full"),
		WithNetworkID(env.Conf.L2.NetworkID),
		WithLogLevel("3"),
	}
	if len(bootNodes) > 0 {
		opts = append(opts, WithBootNode(bootNodes))
	}
	n := NewNode(t, env.Clients.L2, opts...)
	n.TaikoAddr = env.Conf.L2.RollupAddress
	elNode := (*ELNode)(n)
	WaitELNodesUp(env.Context, t, elNode, 10*time.Second)
	return elNode
}

func NewDriverNode(t *hivesim.T, env *TestEnv, l1, l2 *ELNode, enableP2P bool) *Node {
	require.NotNil(t, env.Clients.Driver)
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
	n := NewNode(t, env.Clients.Driver, opts...)
	return n
}

func NewProposerNode(t *hivesim.T, env *TestEnv, l1, l2 *ELNode) *Node {
	require.NotNil(t, env.Clients.Proposer)
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
	return NewNode(t, env.Clients.Proposer, opts...)
}

func NewProverNode(t *hivesim.T, env *TestEnv, l1, l2 *ELNode) *Node {
	require.NotNil(t, env.Clients.Prover)
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
	return NewNode(t, env.Clients.Prover, opts...)
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
