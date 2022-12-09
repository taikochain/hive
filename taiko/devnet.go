package taiko

import (
	"context"
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
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
)

// Devnet is a taiko network with all necessary components, e.g. L1, L2, driver, proposer, prover etc.
type Devnet struct {
	sync.Mutex
	t       *hivesim.T
	C       *Config
	clients *ClientsByRole

	// nodes
	l1Engines []*ELNode
	l2Engines []*ELNode
	drivers   []*Node
	proposers []*Node
	provers   []*Node

	L1Vault *Vault
	L2Vault *Vault

	L2Genesis *core.Genesis
}

func NewDevnet(ctx context.Context, t *hivesim.T) *Devnet {
	d := &Devnet{t: t, C: DefaultConfig}

	clientTypes, err := d.t.Sim.ClientTypes()
	if err != nil {
		d.t.Fatalf("failed to retrieve list of client types: %v", err)
	}
	d.clients = Roles(d.t, clientTypes)
	d.L2Genesis = core.TaikoGenesisBlock(d.C.L2.NetworkID)
	d.L1Vault = NewVault(d.t, d.C.L1.ChainID)
	d.L2Vault = NewVault(d.t, d.C.L2.ChainID)

	l2 := d.AddL2ELNode(ctx, 0, false)
	l1 := d.AddL1ELNode(ctx, 0, l2)
	d.AddDriverNode(ctx, l1, l2, false)
	d.AddProverNode(ctx, l1, l2)
	d.AddProposerNode(ctx, l1, l2)
	return d
}

// AddL1ELNode starts a eth1 image and add it to the network
func (d *Devnet) AddL1ELNode(ctx context.Context, Idx uint, l2 *ELNode) *ELNode {
	if len(d.clients.L1) == 0 {
		d.t.Fatal("no eth1 client types found")
	}
	opts := []Option{
		WithRole("L1Engine"),
		WithL1ChainID(d.C.L1.ChainID),
		WithNetworkID(d.C.L1.NetworkID),
		WithCliquePeriod(d.C.L1.MineInterval),
	}
	n := NewNode(d.t, d.clients.L1[Idx], opts...)
	n.TaikoAddr = d.C.L1.RollupAddress
	elNode := (*ELNode)(n)
	WaitELNodesUp(ctx, d.t, elNode, 10*time.Second)

	l2GenesisHash := GetBlockHashByNumber(ctx, d.t, l2.EthClient(d.t), common.Big0, false)
	d.deployL1Contracts(ctx, elNode, l2GenesisHash)

	d.Lock()
	defer d.Unlock()
	d.l1Engines = append(d.l1Engines, elNode)

	return elNode
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l1Engines) {
		d.t.Fatalf("only have %d L1 nodes, cannot find %d", len(d.l1Engines), idx)
	}
	return d.l1Engines[idx]
}

func (d *Devnet) AddL2ELNode(ctx context.Context, clientIdx uint, enableBootNode bool) *ELNode {
	opts := []Option{
		WithRole("L2Engine"),
		WithJWTSecret(d.C.L2.JWTSecret),
		WithELNodeType("full"),
		WithNetworkID(d.C.L2.NetworkID),
		WithLogLevel("4"),
	}
	if enableBootNode {
		opts = append(opts, WithBootNode(d.getBootNodeParam()))
	}
	n := NewNode(d.t, d.clients.L2[clientIdx], opts...)
	n.TaikoAddr = d.C.L2.RollupAddress
	elNode := (*ELNode)(n)
	WaitELNodesUp(ctx, d.t, elNode, 10*time.Second)

	d.Lock()
	defer d.Unlock()
	d.l2Engines = append(d.l2Engines, elNode)
	return elNode
}

func (d *Devnet) getBootNodeParam() string {
	d.Lock()
	defer d.Unlock()

	urls := make([]string, 0)
	for i, n := range d.l2Engines {
		enodeURL, err := n.EnodeURL()
		if err != nil {
			d.t.Fatalf("failed to get enode url of the %d taiko geth node, error: %v", i, err)
		}
		urls = append(urls, enodeURL)
	}
	return strings.Join(urls, ",")
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l2Engines) {
		d.t.Fatalf("only have %d taiko geth nodes, cannot find %d", len(d.l2Engines), idx)
		return nil
	}
	return d.l2Engines[idx]
}

func (d *Devnet) AddDriverNode(ctx context.Context, l1, l2 *ELNode, enableP2P bool) *Node {
	opts := []Option{
		WithRole("driver"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL2EngineEndpoint(l2.EngineEndpoint()),
		WithL1ContractAddress(d.C.L1.RollupAddress),
		WithL2ContractAddress(d.C.L2.RollupAddress),
		WithThrowawayBlockBuilderPrivateKey(d.C.L2.Throwawayer.PrivateKeyHex),
		WithJWTSecret(d.C.L2.JWTSecret),
	}
	if enableP2P {
		opts = append(opts, WithEnableL2P2P())
	}
	n := NewNode(d.t, d.clients.Driver[0], opts...)
	d.Lock()
	defer d.Unlock()
	d.drivers = append(d.drivers, n)
	return n
}

func (d *Devnet) AddProposerNode(ctx context.Context, l1, l2 *ELNode) *Node {
	if len(d.clients.Proposer) == 0 {
		d.t.Fatalf("no taiko proposer client types found")
	}

	opts := []Option{
		WithRole("proposer"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(d.C.L1.RollupAddress),
		WithL2ContractAddress(d.C.L2.RollupAddress),
		WithProposerPrivateKey(d.C.L2.Proposer.PrivateKeyHex),
		WithSuggestedFeeRecipient(d.C.L2.SuggestedFeeRecipient.Address),
		WithProposeInterval(d.C.L2.ProposeInterval),
	}

	if d.C.L2.ProduceInvalidBlocksInterval != 0 {
		opts = append(opts, WithProduceInvalidBlocksInterval(d.C.L2.ProduceInvalidBlocksInterval))
	}
	n := NewNode(d.t, d.clients.Proposer[0], opts...)
	d.Lock()
	defer d.Unlock()
	d.proposers = append(d.proposers, n)
	return n

}

func (d *Devnet) AddProverNode(ctx context.Context, l1, l2 *ELNode) *Node {
	if len(d.clients.Prover) == 0 {
		d.t.Fatalf("no taiko prover client types found")
	}
	if err := d.addWhitelist(ctx, l1.EthClient(d.t)); err != nil {
		d.t.Fatalf("add whitelist failed, err=%v", err)
	}
	opts := []Option{
		WithRole("prover"),
		WithNoCheck(),
		WithL1Endpoint(l1.WsRpcEndpoint()),
		WithL2Endpoint(l2.WsRpcEndpoint()),
		WithL1ContractAddress(d.C.L1.RollupAddress),
		WithL2ContractAddress(d.C.L2.RollupAddress),
		WithProverPrivateKey(d.C.L2.Prover.PrivateKeyHex),
	}
	n := NewNode(d.t, d.clients.Prover[0], opts...)
	d.Lock()
	defer d.Unlock()
	d.provers = append(d.provers, n)
	return n
}

func (d *Devnet) addWhitelist(ctx context.Context, cli *ethclient.Client) error {
	taikoL1, err := bindings.NewTaikoL1Client(d.C.L1.RollupAddress, cli)
	if err != nil {
		return err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(d.C.L1.Deployer.PrivateKey, d.C.L1.ChainID)
	if err != nil {
		return err
	}
	opts.GasTipCap = big.NewInt(1500000000)
	tx, err := taikoL1.WhitelistProver(opts, d.C.L2.Prover.Address, true)
	if err != nil {
		return err
	}

	receipt, err := rpc.WaitReceipt(ctx, cli, tx)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		d.t.Fatal("Failed to commit transactions list", "txHash", receipt.TxHash)
	}

	d.t.Log("Add prover to whitelist finished", "height", receipt.BlockNumber)

	return nil
}

// deployL1Contracts runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func (d *Devnet) deployL1Contracts(ctx context.Context, l1Node *ELNode, l2GenesisHash common.Hash) {
	if d.clients.Contract == nil {
		d.t.Fatalf("no taiko protocol client types found")
	}
	opts := []Option{
		WithNoCheck(),
		WithPrivateKey(d.C.L1.Deployer.PrivateKeyHex),
		WithL1DeployerAddress(d.C.L1.Deployer.Address),
		WithL2GenesisBlockHash(l2GenesisHash),
		WithL2ContractAddress(d.C.L2.RollupAddress),
		WithMainnetUrl(l1Node.HttpRpcEndpoint()),
		WithL2ChainID(d.C.L2.ChainID),
	}
	n := NewNode(d.t, d.clients.Contract, opts...)
	result, err := n.Exec("deploy.sh")
	if err != nil || result.ExitCode != 0 {
		d.t.Fatalf("failed to deploy contract on engine node %s, error: %v, result: %v",
			l1Node.Container, err, result)
	}
	d.t.Logf("Deploy contracts on %s %s(%s)", l1Node.Type, l1Node.Container, l1Node.IP)
}
