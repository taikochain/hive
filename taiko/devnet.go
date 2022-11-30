package taiko

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/hive/hivesim"
)

// L2 combines the taiko geth and taiko driver
type L2 struct {
	Geth   *ELNode
	Driver *DriverNode
}

// Devnet is a taiko network with all necessary components, e.g. l1, l2, driver, proposer, prover etc.
type Devnet struct {
	sync.Mutex

	t       *hivesim.T
	clients *ClientsByRole

	// nodes
	contract  *ContractNode // contracts deploy client
	l1Engines []*ELNode
	l2Engines []*ELNode
	drivers   []*DriverNode
	proposers []*ProposerNode
	provers   []*ProverNode

	L1Vault *Vault
	L2Vault *Vault

	accounts *Accounts // all accounts will be used in the network

	deployments *Deployments // contracts deployment info
	bindings    *Bindings    // bindings of contracts

	L1Cfg *core.Genesis
	L2Cfg *core.Genesis

	config *Config // network config
}

func NewDevnet(t *hivesim.T) *Devnet {
	clientTypes, err := t.Sim.ClientTypes()
	if err != nil {
		t.Fatalf("failed to retrieve list of client types: %v", err)
	}

	roles := Roles(t, clientTypes)
	t.Logf("creating devnet with roles: %s", roles)
	d := &Devnet{
		t:       t,
		clients: roles,

		accounts:    DefaultAccounts(t),
		deployments: DefaultDeployments,
		config:      DefaultConfig,
		L1Cfg:       new(core.Genesis),
		L2Cfg:       core.TaikoGenesisBlock(DefaultConfig.L2NetworkID),
	}

	data, err := ioutil.ReadFile("/genesis.json")
	if err != nil {
		d.t.Fatal("can not read l1 genesis file", "err", err)
	}
	if err := json.Unmarshal(data, d.L1Cfg); err != nil {
		d.t.Fatal("can not load l1 genesis", "err", err)
	}
	d.L1Vault = NewVault(t, d.L1Cfg.Config)
	d.L2Vault = NewVault(t, d.L2Cfg.Config)
	return d
}

// Init initializes the network
func (d *Devnet) Init() {
}

// StartL1ELNodes starts a eth1 image and add it to the network
func (d *Devnet) StartL1ELNodes(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.L1) == 0 {
		d.t.Fatal("no eth1 client types found")
	}
	opts = append(opts, hivesim.Params{
		envTaikoL1ChainId:      d.config.L1ChainID.String(),
		envTaikoL1CliquePeriod: strconv.FormatUint(d.config.L1MineInterval, 10),
	})

	for i, c := range d.clients.L1 {
		name := fmt.Sprintf("%s:%d", c.Name, i)
		d.l1Engines = append(d.l1Engines, &ELNode{d.t.StartClient(name, opts...)})
	}
	WaitELNodesUp(ctx, d.t, d.l1Engines, 10*time.Second)
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l1Engines) {
		d.t.Fatalf("only have %d l1 nodes, cannot find %d", len(d.l1Engines), idx)
		return nil
	}
	return d.l1Engines[idx]
}

func (d *Devnet) WaitL1Block(ctx context.Context, idx int, atLeastHight uint64) error {
	d.Lock()
	defer d.Unlock()

	return WaitBlock(ctx, d.GetL1ELNode(idx).EthClient(), atLeastHight)
}

func (d *Devnet) StartL2ELNodes(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()
	gethOpts := append(opts, hivesim.Params{
		envTaikoNetworkId: strconv.FormatUint(d.config.L2NetworkID, 10),
	})
	for i, c := range d.clients.L2 {
		name := fmt.Sprintf("%s:%d", c.Name, i)
		if i > 0 {
			l2 := d.GetL2ELNode(0)
			enodeURL, err := l2.EnodeURL()
			if err != nil {
				d.t.Fatalf("failed to get enode url of the first taiko geth node, error: %w", err)
			}
			gethOpts = append(gethOpts, hivesim.Params{
				envTaikoBootNode: enodeURL,
			})
		}
		d.l2Engines = append(d.l2Engines, &ELNode{d.t.StartClient(name, gethOpts...)})
	}
	WaitELNodesUp(ctx, d.t, d.l2Engines, 10*time.Second)
}

func (d *Devnet) StartDriverNodes(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	for i, c := range d.clients.Driver {
		name := fmt.Sprintf("%s:%d", c.Name, i)
		l2Node := d.GetL2ELNode(i)
		// start taiko-driver
		l1 := d.GetL1ELNode(0)
		driverOpts := append(opts, hivesim.Params{
			envTaikoL1RPCEndpoint:                   l1.WsRpcEndpoint(),
			envTaikoL2RPCEndpoint:                   l2Node.WsRpcEndpoint(),
			envTaikoL2EngineEndpoint:                l2Node.EngineEndpoint(),
			envTaikoL1RollupAddress:                 d.deployments.L1RollupAddress.Hex(),
			envTaikoL2RollupAddress:                 d.deployments.L2RollupAddress.Hex(),
			envTaikoThrowawayBlockBuilderPrivateKey: d.accounts.Throwawayer.PrivateKeyHex,
			"HIVE_CHECK_LIVE_PORT":                  "0",
		})
		d.drivers = append(d.drivers, &DriverNode{d.t.StartClient(name, driverOpts...)})
	}
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l2Engines) {
		d.t.Fatalf("only have %d taiko geth nodes, cannot find %d", len(d.l2Engines), idx)
		return nil
	}
	return d.l2Engines[idx]
}

func (d *Devnet) StartProposerNodes(ctx context.Context, params *PipelineParams) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Proposer) == 0 {
		d.t.Fatalf("no taiko proposer client types found")
	}
	for i, n := range d.clients.Proposer {
		l1 := d.GetL1ELNode(i % len(d.l1Engines))
		l2 := d.GetL2ELNode(i % len(d.l2Engines))
		var opts []hivesim.StartOption
		opts = append(opts, hivesim.Params{
			envTaikoL1RPCEndpoint:         l1.WsRpcEndpoint(),
			envTaikoL2RPCEndpoint:         l2.WsRpcEndpoint(),
			envTaikoL1RollupAddress:       d.deployments.L1RollupAddress.Hex(),
			envTaikoL2RollupAddress:       d.deployments.L2RollupAddress.Hex(),
			envTaikoProposerPrivateKey:    d.accounts.Proposer.PrivateKeyHex,
			envTaikoSuggestedFeeRecipient: d.accounts.SuggestedFeeRecipient.Address.Hex(),
			envTaikoProposeInterval:       d.config.ProposeInterval.String(),
			"HIVE_CHECK_LIVE_PORT":        "0",
		},
		)
		if params != nil && params.ProduceInvalidBlocksInterval != 0 {
			opts = append(opts, hivesim.Params{
				envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(params.ProduceInvalidBlocksInterval, 10),
			})
		}
		d.proposers = append(d.proposers, &ProposerNode{d.t.StartClient(n.Name, opts...)})
	}
}

func (d *Devnet) StartProverNodes(ctx context.Context) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Prover) == 0 {
		d.t.Fatalf("no taiko prover client types found")
		return
	}
	for i, n := range d.clients.Prover {
		l1 := d.GetL1ELNode(i % len(d.l1Engines))
		l2 := d.GetL2ELNode(i % len(d.l2Engines))
		var opts []hivesim.StartOption
		opts = append(opts, hivesim.Params{
			envTaikoL1RPCEndpoint:    l1.WsRpcEndpoint(),
			envTaikoL2RPCEndpoint:    l2.WsRpcEndpoint(),
			envTaikoL1RollupAddress:  d.deployments.L1RollupAddress.Hex(),
			envTaikoL2RollupAddress:  d.deployments.L2RollupAddress.Hex(),
			envTaikoProverPrivateKey: d.accounts.Prover.PrivateKeyHex,
			"HIVE_CHECK_LIVE_PORT":   "0",
		})
		d.provers = append(d.provers, &ProverNode{d.t.StartClient(n.Name, opts...)})
	}

}

// DeployL1Contracts runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func (d *Devnet) DeployL1Contracts(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if d.clients.Contract == nil {
		d.t.Fatalf("no taiko protocol client types found")
	}
	for i, e := range d.l1Engines {
		opts = append(opts, hivesim.Params{
			envTaikoL1DeployerAddress:  d.accounts.L1Deployer.Address.Hex(),
			envTaikoL2GenesisBlockHash: d.deployments.L2GenesisBlockHash.Hex(),
			envTaikoL2RollupAddress:    d.deployments.L2RollupAddress.Hex(),
			envTaikoMainnetUrl:         e.HttpRpcEndpoint(),
			envTaikoPrivateKey:         d.accounts.L1Deployer.PrivateKeyHex,
			envTaikoL2ChainId:          d.config.L2ChainID.String(),
			"HIVE_CHECK_LIVE_PORT":     "0",
		})

		n := &ContractNode{d.t.StartClient(d.clients.Contract.Name, opts...)}
		result, err := n.Exec("deploy.sh")
		if err != nil || result.ExitCode != 0 {
			d.t.Fatalf("failed to deploy contract on engine node %d(%s), error: %v, result: %v",
				i, e.Container, err, result)
		}
		d.t.Sim.StopClient(d.t.SuiteID, d.t.TestID, n.Container)
	}
}

type PipelineParams struct {
	ProduceInvalidBlocksInterval uint64
}

// StartDevnetWithSingleInstance each component runs only one instance
func StartDevnetWithSingleInstance(ctx context.Context, d *Devnet, params *PipelineParams) error {
	d.Init()
	d.StartL1ELNodes(ctx)
	d.StartL2ELNodes(ctx)
	d.DeployL1Contracts(ctx)
	// TODO(alex):deploy l2 contracts
	d.StartProposerNodes(ctx, params)
	d.StartDriverNodes(ctx)

	d.StartProverNodes(ctx)
	// init bindings for tests
	d.InitBindingsL1(0)
	d.InitBindingsL2(0)
	return d.WaitL1Block(ctx, 0, 2)
}
