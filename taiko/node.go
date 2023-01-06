package taiko

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
)

type Node struct {
	*hivesim.Client
	role      string
	opts      []hivesim.StartOption
	TaikoAddr common.Address
}

type NodeOption func(*Node)

func NewNode(t *hivesim.T, c *hivesim.ClientDefinition, opts ...NodeOption) *Node {
	n := new(Node)
	for _, o := range opts {
		o(n)
	}
	n.Client = t.StartClient(c.Name, n.opts...)
	return n
}

// NewL1ELNode starts a eth1 image and add it to the network
func (e *TestEnv) NewL1ELNode(opts ...NodeOption) *ELNode {
	require.NotNil(e.T, e.Clients.L1)
	opts = append(opts,
		WithRole("L1Engine"),
		WithL1ChainID(e.Conf.L1.ChainID),
		WithNetworkID(e.Conf.L1.NetworkID),
		WithCliquePeriod(e.Conf.L1.MineInterval),
	)
	n := NewNode(e.T, e.Clients.L1, opts...)
	n.TaikoAddr = e.Conf.L1.RollupAddress
	elNode := (*ELNode)(n)
	e.WaitELNodesUp(elNode, 10*time.Second)
	return elNode
}

// deployL1Contracts runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func (e *TestEnv) deployL1Contracts(l1Node, l2 *ELNode) {
	require.NotNil(e.T, e.Clients.Contract)
	l2GenesisHash := e.GetBlockHashByNumber(l2, common.Big0, false)
	opts := []NodeOption{
		WithNoCheck(),
		WithPrivateKey(e.Conf.L1.Deployer.PrivateKeyHex),
		WithL1DeployerAddress(e.Conf.L1.Deployer.Address),
		WithL2GenesisBlockHash(l2GenesisHash),
		WithL2ContractAddress(e.Conf.L2.RollupAddress),
		WithMainnetUrl(l1Node.HttpRpcEndpoint()),
		WithL2ChainID(e.Conf.L2.ChainID),
	}
	n := NewNode(e.T, e.Clients.Contract, opts...)
	result, err := n.Exec("deploy.sh")
	if err != nil || result.ExitCode != 0 {
		e.T.Fatalf("failed to deploy contract on engine node %s, error: %v, result: %v",
			l1Node.Container, err, result)
	}
	e.T.Logf("Deploy contracts on %s %s(%s)", l1Node.Type, l1Node.Container, l1Node.IP)
}

func (e *TestEnv) NewFullSyncL2ELNode(opts ...NodeOption) *ELNode {
	opts = append(opts, WithELNodeType("full"))
	return e.NewL2ELNode(opts...)
}

func (e *TestEnv) NewL2ELNode(opts ...NodeOption) *ELNode {
	require.NotNil(e.T, e.Clients.L2)
	opts = append(opts,
		WithRole("L2Engine"),
		WithJWTSecret(e.Conf.L2.JWTSecret),
		WithNetworkID(e.Conf.L2.NetworkID),
		WithLogLevel("3"),
	)
	n := NewNode(e.T, e.Clients.L2, opts...)
	n.TaikoAddr = e.Conf.L2.RollupAddress
	elNode := (*ELNode)(n)
	e.WaitELNodesUp(elNode, 10*time.Second)
	return elNode
}

func (e *TestEnv) NewDriverNode(l1, l2 *ELNode, opts ...NodeOption) *Node {
	require.NotNil(e.T, e.Clients.Driver)
	opts = append(opts,
		WithRole("driver"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL2EngineEndpoint(l2.EngineEndpoint()),
		WithL1ContractAddress(e.Conf.L1.RollupAddress),
		WithL2ContractAddress(e.Conf.L2.RollupAddress),
		WithThrowawayBlockBuilderPrivateKey(e.Conf.L2.Throwawayer.PrivateKeyHex),
		WithJWTSecret(e.Conf.L2.JWTSecret),
	)
	n := NewNode(e.T, e.Clients.Driver, opts...)
	return n
}

func (e *TestEnv) NewProposerNode(l1, l2 *ELNode, opts ...NodeOption) *Node {
	require.NotNil(e.T, e.Clients.Proposer)
	opts = append(opts,
		WithRole("proposer"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(e.Conf.L1.RollupAddress),
		WithL2ContractAddress(e.Conf.L2.RollupAddress),
		WithProposerPrivateKey(e.Conf.L2.Proposer.PrivateKeyHex),
		WithSuggestedFeeRecipient(e.Conf.L2.SuggestedFeeRecipient.Address),
		WithProposeInterval(e.Conf.L2.ProposeInterval),
	)
	if e.Conf.L2.ProduceInvalidBlocksInterval != 0 {
		opts = append(opts, WithProduceInvalidBlocksInterval(e.Conf.L2.ProduceInvalidBlocksInterval))
	}
	return NewNode(e.T, e.Clients.Proposer, opts...)
}

func (e *TestEnv) NewProverNode(l1, l2 *ELNode, opts ...NodeOption) *Node {
	require.NotNil(e.T, e.Clients.Prover)
	opts = append(opts,
		WithRole("prover"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(e.Conf.L1.RollupAddress),
		WithL2ContractAddress(e.Conf.L2.RollupAddress),
		WithProverPrivateKey(e.Conf.L2.Prover.PrivateKeyHex),
	)
	return NewNode(e.T, e.Clients.Prover, opts...)
}
