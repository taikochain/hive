package taiko

import (
	"context"
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

func (l2 *L2) Genesis() *core.Genesis {
	return core.TaikoGenesisBlock()
}

// Devnet is a taiko network with all necessary components, e.g. l1, l2, driver, proposer, prover etc.
type Devnet struct {
	sync.Mutex

	t       *hivesim.T
	clients *ClientsByRole

	l1s      []*ELNode     // L1 nodes
	l2s      []*L2         // L2 nodes
	contract *ContractNode // contracts deploy client
	// TODO(alex): multi proposer and prover
	proposer *ProposerNode // proposer client
	prover   *ProverNode   // prover client

	L1Vault *Vault // l1 vault
	L2Vault *Vault // l2 vault

	accounts *Accounts // all accounts will be used in the network

	deployments *Deployments // contracts deployment info
	bindings    *Bindings    // bindings of contracts

	config *Config // network config
}

func NewDevnet(t *hivesim.T) *Devnet {
	clientTypes, err := t.Sim.ClientTypes()
	if err != nil {
		t.Fatalf("failed to retrieve list of client types: %v", err)
	}

	roles := Roles(clientTypes)
	t.Logf("creating devnet with roles: %s", roles)
	return &Devnet{
		t:       t,
		clients: roles,

		L1Vault: NewVault(t, DefaultConfig.L1ChainID),
		L2Vault: NewVault(t, DefaultConfig.L2ChainID),

		accounts:    DefaultAccounts(t),
		deployments: DefaultDeployments,
		config:      DefaultConfig,
	}
}

// Init initializes the network
func (d *Devnet) Init() {
}

// AddL1 starts a eth1 image and add it to the network
func (d *Devnet) AddL1(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.L1) == 0 {
		d.t.Fatal("no eth1 client types found")
		return
	}
	opts = append(opts, hivesim.Params{
		envTaikoL1ChainId:      d.config.L1ChainID.String(),
		envTaikoL1CliquePeriod: strconv.FormatUint(d.config.L1MineInterval, 10),
	})
	d.l1s = append(d.l1s, &ELNode{d.t.StartClient(d.clients.L1[0].Name, opts...)})
}

func (d *Devnet) GetL1(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l1s) {
		d.t.Fatalf("only have %d l1 nodes, cannot find %d", len(d.l1s), idx)
		return nil
	}
	return d.l1s[idx]
}

func (d *Devnet) WaitUpL1(ctx context.Context, idx int, timeout time.Duration) {
	d.Lock()
	defer d.Unlock()

	WaitUp(ctx, d.t, d.GetL1(idx).EthClient(), timeout)
}

func (d *Devnet) WaitL1Block(ctx context.Context, idx int, atLeastHight uint64) error {
	d.Lock()
	defer d.Unlock()

	return WaitBlock(ctx, d.GetL1(idx).EthClient(), atLeastHight)
}

func (d *Devnet) AddProtocol(ctx context.Context, idx int, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if d.clients.Contract == nil {
		d.t.Fatalf("no taiko protocol client types found")
		return
	}
	l1 := d.GetL1(idx)
	opts = append(opts, hivesim.Params{
		envTaikoL1DeployerAddress:  d.accounts.L1Deployer.Address.Hex(),
		envTaikoL2GenesisBlockHash: d.deployments.L2GenesisBlockHash.Hex(),
		envTaikoL2RollupAddress:    d.deployments.L2RollupAddress.Hex(),
		envTaikoMainnetUrl:         l1.HttpRpcEndpoint(),
		envTaikoPrivateKey:         d.accounts.L1Deployer.PrivateKeyHex,
		envTaikoL2ChainId:          d.config.L2ChainID.String(),
		"HIVE_CHECK_LIVE_PORT":     "0",
	})

	d.contract = &ContractNode{d.t.StartClient(d.clients.Contract.Name, opts...)}
}

func (d *Devnet) AddL2(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.L2) == 0 {
		d.t.Fatal("no taiko geth client types found")
		return
	}

	if len(d.clients.Driver) == 0 {
		d.t.Fatal("no taiko driver client types found")
		return
	}

	// start taiko-geth
	gethOpts := append(opts, hivesim.Params{
		envTaikoNetworkId: strconv.FormatUint(d.config.L2NetworkID, 10),
	})
	if len(d.l2s) > 1 {
		l2 := d.GetL2(0)
		enodeURL, err := l2.Geth.EnodeURL()
		if err != nil {
			d.t.Fatalf("failed to get enode url of the first taiko geth node, error: %w", err)
			return
		}
		// set bootnode
		gethOpts = append(gethOpts, hivesim.Params{
			envTaikoBootNode: enodeURL,
		})
	}
	geth := &ELNode{d.t.StartClient(d.clients.L2[0].Name, gethOpts...)}
	// start taiko-driver
	l1 := d.GetL1(0)
	driverOpts := append(opts, hivesim.Params{
		envTaikoL1RPCEndpoint:                   l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:                   geth.WsRpcEndpoint(),
		envTaikoL2EngineEndpoint:                geth.EngineEndpoint(),
		envTaikoL1RollupAddress:                 d.deployments.L1RollupAddress.Hex(),
		envTaikoL2RollupAddress:                 d.deployments.L2RollupAddress.Hex(),
		envTaikoThrowawayBlockBuilderPrivateKey: d.accounts.Throwawayer.PrivateKeyHex,
		"HIVE_CHECK_LIVE_PORT":                  "0",
	})
	driver := &DriverNode{d.t.StartClient(d.clients.Driver[0].Name, driverOpts...)}

	d.l2s = append(d.l2s, &L2{
		Geth:   geth,
		Driver: driver,
	})
}

func (d *Devnet) GetL2(idx int) *L2 {
	if idx < 0 || idx >= len(d.l2s) {
		d.t.Fatalf("only have %d taiko geth nodes, cannot find %d", len(d.l2s), idx)
		return nil
	}
	return d.l2s[idx]
}

func (d *Devnet) WaitUpL2(ctx context.Context, idx int, timeout time.Duration) {
	d.Lock()
	defer d.Unlock()

	WaitUp(ctx, d.t, d.GetL2(idx).Geth.EthClient(), timeout)
}

func (d *Devnet) AddProposer(ctx context.Context, l1Idx, l2Idx int, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Proposer) == 0 {
		d.t.Fatalf("no taiko proposer client types found")
		return
	}
	l1 := d.GetL1(l1Idx)
	l2 := d.GetL2(l2Idx)
	opts = append(opts, hivesim.Params{
		envTaikoL1RPCEndpoint:         l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:         l2.Geth.WsRpcEndpoint(),
		envTaikoL1RollupAddress:       d.deployments.L1RollupAddress.Hex(),
		envTaikoL2RollupAddress:       d.deployments.L2RollupAddress.Hex(),
		envTaikoProposerPrivateKey:    d.accounts.Proposer.PrivateKeyHex,
		envTaikoSuggestedFeeRecipient: d.accounts.SuggestedFeeRecipient.Address.Hex(),
		envTaikoProposeInterval:       d.config.ProposeInterval.String(),
		"HIVE_CHECK_LIVE_PORT":        "0",
	})
	d.proposer = &ProposerNode{d.t.StartClient(d.clients.Proposer[0].Name, opts...)}
}

func (d *Devnet) AddProver(ctx context.Context, l1Idx, l2Idx int, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Prover) == 0 {
		d.t.Fatalf("no taiko prover client types found")
		return
	}
	l1 := d.GetL1(l1Idx)
	l2 := d.GetL2(l2Idx)
	opts = append(opts, hivesim.Params{
		envTaikoL1RPCEndpoint:    l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:    l2.Geth.WsRpcEndpoint(),
		envTaikoL1RollupAddress:  d.deployments.L1RollupAddress.Hex(),
		envTaikoL2RollupAddress:  d.deployments.L2RollupAddress.Hex(),
		envTaikoProverPrivateKey: d.accounts.Prover.PrivateKeyHex,
		"HIVE_CHECK_LIVE_PORT":   "0",
	})
	d.prover = &ProverNode{d.t.StartClient(d.clients.Prover[0].Name, opts...)}
}

// RunDeployL1 runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func (d *Devnet) RunDeployL1(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	result, err := d.contract.Exec("deploy.sh")
	if err != nil || result.ExitCode != 0 {
		d.t.Fatalf("failed to run deploy_l1 error: %v, result: %v", err, result)
		return
	}
}

type PipelineParams struct {
	ProduceInvalidBlocksInterval uint64
}

// StartDevnetWithSingleInstance each component runs only one instance
func StartDevnetWithSingleInstance(ctx context.Context, d *Devnet, params *PipelineParams) error {
	d.Init()
	// start l1 node
	d.AddL1(ctx)
	d.WaitUpL1(ctx, 0, 10*time.Second)
	// deploy l1 contracts
	d.AddProtocol(ctx, 0)
	d.RunDeployL1(ctx)
	// start l2
	d.AddL2(ctx)
	d.WaitUpL2(ctx, 0, 10*time.Second)
	// add components
	if params != nil && params.ProduceInvalidBlocksInterval != 0 {
		d.AddProposer(ctx, 0, 0, hivesim.Params{
			envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(params.ProduceInvalidBlocksInterval, 10),
		})
	} else {
		d.AddProposer(ctx, 0, 0)
	}
	d.AddProver(ctx, 0, 0)
	// init bindings for tests
	d.InitBindingsL1(0)
	d.InitBindingsL2(0)
	return d.WaitL1Block(ctx, 0, 2)
}
