package taiko

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/hive/hivesim"
)

// L2 combines the taiko geth and taiko driver
type L2 struct {
	Geth   *Node
	Driver *Node
}

func (l2 *L2) Genesis() *core.Genesis {
	return core.TaikoGenesisBlock()
}

// Devnet is a taiko network with all necessary components, e.g. l1, l2, driver, proposer, prover etc.
type Devnet struct {
	sync.Mutex

	t       *hivesim.T
	clients *Clients

	l1s []*Node // L1 nodes
	l2s []*L2   // L2 nodes

	protocol *Node // protocol client
	proposer *Node // proposer client
	prover   *Node // prover client

	L1Vault *Vault // l1 vault
	L2Vault *Vault // l2 vault

	accounts    *Accounts    // all accounts will be used in the network
	deployments *Deployments // contracts deployment info
	bindings    *Bindings    // bindings of contracts

	config *Config // network config
}

func NewDevnet(t *hivesim.T) *Devnet {
	clientTypes, err := t.Sim.ClientTypes()
	if err != nil {
		t.Fatalf("failed to retrieve list of client types: %v", err)
	}

	clients := NewClients(clientTypes)
	t.Logf("creating devnet with clients: %s", clients)
	return &Devnet{
		t:       t,
		clients: clients,

		L1Vault:     NewVault(t, DefaultConfig.L1ChainID),
		L2Vault:     NewVault(t, DefaultConfig.L2ChainID),
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
	d.l1s = append(d.l1s, NewNode(d.clients.L1[0].Name, d.t, opts...))
}

func (d *Devnet) GetL1(idx int) *Node {
	if idx < 0 || idx >= len(d.l1s) {
		d.t.Fatalf("only have %d l1 nodes, cannot find %d", len(d.l1s), idx)
		return nil
	}
	return d.l1s[idx]
}

func (d *Devnet) WaitUpL1(ctx context.Context, idx int, timeout time.Duration) {
	d.Lock()
	defer d.Unlock()

	WaitUp(ctx, d.t, d.GetL1(idx).Eth, timeout)
}

func (d *Devnet) WaitL1Block(ctx context.Context, idx int, atLeastHight uint64) error {
	d.Lock()
	defer d.Unlock()

	return WaitBlock(ctx, d.GetL1(idx).Eth, atLeastHight)
}

func (d *Devnet) AddProtocol(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.TaikoProtocol) == 0 {
		d.t.Fatalf("no taiko protocol client types found")
		return
	}

	opts = append(opts, hivesim.Params{
		"HIVE_CHECK_LIVE_PORT": "0",
	})

	d.protocol = NewNode(d.clients.TaikoProtocol[0].Name, d.t, opts...)
}

func (d *Devnet) AddL2(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.TaikoGeth) == 0 {
		d.t.Fatal("no taiko geth client types found")
		return
	}

	if len(d.clients.TaikoDriver) == 0 {
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
	geth := NewNode(d.clients.TaikoGeth[0].Name, d.t, gethOpts...)
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
	driver := NewNode(d.clients.TaikoDriver[0].Name, d.t, driverOpts...)

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

	WaitUp(ctx, d.t, d.GetL2(idx).Geth.Eth, timeout)
}

func (d *Devnet) AddProposer(ctx context.Context, l1Idx, l2Idx int, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.TaikoProposer) == 0 {
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
	d.proposer = NewNode(d.clients.TaikoProposer[0].Name, d.t, opts...)
}

func (d *Devnet) AddProver(ctx context.Context, l1Idx, l2Idx int, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.TaikoProver) == 0 {
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
	d.prover = NewNode(d.clients.TaikoProver[0].Name, d.t, opts...)
}

// RunDeployL1 runs the `npx hardhat deploy_l1` command in `taiko-protocol` container
func (d *Devnet) RunDeployL1(ctx context.Context, idx int, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	l1 := d.GetL1(idx)

	result, err := d.protocol.Exec(
		"/bin/sh", "-c",
		fmt.Sprintf("MAINNET_URL=%s", l1.HttpRpcEndpoint()),
		fmt.Sprintf("PRIVATE_KEY=%s", d.accounts.L1Deployer.PrivateKeyHex),
		"npx", "hardhat", "deploy_l1",
		"--network", "mainnet",
		"--dao-vault", d.accounts.L1Deployer.Address.Hex(),
		"--team-vault", d.accounts.L1Deployer.Address.Hex(),
		"--l2-genesis-block-hash", d.deployments.L2GenesisBlockHash.Hex(),
		"--v1-taiko-l2", d.deployments.L2RollupAddress.Hex(),
		"--confirmations", "1",
	)
	if err != nil || result.ExitCode != 0 {
		d.t.Fatalf("failed to run deploy_l1 error: %v, result: %v", err, result)
		return
	}
}
