package taiko

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
)

type Node struct {
	*hivesim.Client
	role string
	opts []hivesim.StartOption
}

type deployResult struct {
	rollupAddress    common.Address
	bridgeAddress    common.Address
	vaultAddress     common.Address
	testERC20Address common.Address
}

type NodeOption func(*Node)

func NewNode(t *hivesim.T, c *hivesim.ClientDefinition, opts ...NodeOption) *Node {
	require.NotNil(t, c)
	n := new(Node)
	for _, o := range opts {
		o(n)
	}
	n.Client = t.StartClient(c.Name, n.opts...)
	return n
}

// NewL1ELNode starts a eth1 image and add it to the network
func (e *TestEnv) NewL1ELNode(l2 *ELNode, opts ...NodeOption) *ELNode {
	t, c, def := e.T, e.Conf, e.Clients.L1
	opts = append(opts,
		WithRole("L1Engine"),
		WithL1ChainID(c.L1.ChainID),
		WithNetworkID(c.L1.NetworkID),
		WithCliquePeriod(c.L1.MineInterval),
	)
	n := NewNode(t, def, opts...)
	l1 := &ELNode{Node: n}
	e.deployL1Contracts(l1, l2)
	return l1
}

// deployL1Contracts runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func (e *TestEnv) deployL1Contracts(l1, l2 *ELNode) {
	t, c, def := e.T, e.Conf, e.Clients.Contract
	opts := []NodeOption{
		WithNoCheck(),
		WithPrivateKey(c.L1.Deployer.PrivateKeyHex),
		WithL1DeployerAddress(c.L1.Deployer.Address),
		WithL2GenesisBlockHash(l2.genesisHash),
		WithL2ContractAddress(l2.deploy.rollupAddress),
		WithMainnetUrl(l1.HttpRpcEndpoint()),
		WithL2ChainID(c.L2.ChainID),
	}
	t.Log("start deploy contracts on L1")
	n := NewNode(t, def, opts...)
	result, err := n.Exec("deploy.sh")
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("failed to deploy contract on engine node %s, error: %v, result: %+v",
			l1.Container, err, result)
	}
	t.Logf("Deploy contracts on %s %s(%s)", l1.Type, l1.Container, l1.IP)
	t.Log("Deploy result begin")
	t.Log(result.Stdout)
	t.Log("Deploy result end")
	l1.deploy = n.getL1Deployments(t)
}

func (n *Node) getL1Deployments(t *hivesim.T) *deployResult {
	query := func(key string) common.Address {
		result, err := n.Exec("deploy_result.sh", key)
		if err != nil || result.ExitCode != 0 {
			t.Fatalf("failed to get deploy result on L1 engine node %s, error: %v, result: %+v",
				n.Container, err, result)
		}
		return common.HexToAddress(strings.TrimSpace(result.Stdout))
	}
	return &deployResult{
		rollupAddress: query(".contracts.TaikoL1"),
		bridgeAddress: query(".contracts.Bridge"),
		vaultAddress:  query(".contracts.TokenVault"),
	}
}

func (e *TestEnv) NewFullSyncL2ELNode(opts ...NodeOption) *ELNode {
	opts = append(opts, WithELNodeType("full"))
	return e.NewL2ELNode(opts...)
}

func (e *TestEnv) NewL2ELNode(opts ...NodeOption) *ELNode {
	t, ctx, c, def := e.T, e.Context, e.Conf, e.Clients.L2
	opts = append(opts,
		WithRole("L2Engine"),
		WithJWTSecret(c.L2.JWTSecret),
		WithNetworkID(c.L2.NetworkID),
		WithLogLevel("3"),
	)
	n := NewNode(t, def, opts...)
	l2 := &ELNode{Node: n}
	l2.deploy = l2.getL2Deployments(t)
	g, err := GetBlockHashByNumber(ctx, l2, common.Big0, false)
	require.NoError(t, err)
	l2.genesisHash = g
	return l2
}

func (e *TestEnv) NewDriverNode(l1, l2 *ELNode, opts ...NodeOption) *Node {
	t, c, def := e.T, e.Conf, e.Clients.Driver
	opts = append(opts,
		WithRole("driver"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL2EngineEndpoint(l2.EngineEndpoint()),
		WithL1ContractAddress(l1.deploy.rollupAddress),
		WithL2ContractAddress(l2.deploy.rollupAddress),
		WithThrowawayBlockBuilderPrivateKey(c.L2.Throwawayer.PrivateKeyHex),
		WithJWTSecret(c.L2.JWTSecret),
	)
	n := NewNode(t, def, opts...)
	return n
}

func (e *TestEnv) NewProposerNode(l1, l2 *ELNode, opts ...NodeOption) *Node {
	t, c, def := e.T, e.Conf, e.Clients.Proposer
	opts = append(opts,
		WithRole("proposer"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(l1.deploy.rollupAddress),
		WithL2ContractAddress(l2.deploy.rollupAddress),
		WithProposerPrivateKey(c.L2.Proposer.PrivateKeyHex),
		WithSuggestedFeeRecipient(c.L2.SuggestedFeeRecipient.Address),
		WithProposeInterval(c.L2.ProposeInterval),
	)
	return NewNode(t, def, opts...)
}

func (e *TestEnv) NewProverNode(l1, l2 *ELNode, opts ...NodeOption) *Node {
	t, c, def := e.T, e.Conf, e.Clients.Prover
	opts = append(opts,
		WithRole("prover"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(l1.deploy.rollupAddress),
		WithL2ContractAddress(l2.deploy.rollupAddress),
		WithProverPrivateKey(c.L2.Prover.PrivateKeyHex),
	)
	return NewNode(t, def, opts...)
}
