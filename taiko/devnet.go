package taiko

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
)

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

	L1Genesis *core.Genesis
	L2Genesis *core.Genesis

	rollupConf *RollupConfig
	nodesConf  *NodesConfig
}

func NewDevnet(t *hivesim.T, conf *NodesConfig) *Devnet {
	d := &Devnet{
		t:           t,
		nodesConf:   conf,
		deployments: DefaultDeployments,
		rollupConf:  DefaultRollupConfig,
		accounts:    DefaultAccounts(t),
	}
	return d
}

func (d *Devnet) Start(ctx context.Context) error {
	d.Init()
	d.StartL1ELNodes(ctx)
	d.StartL2ELNodes(ctx)
	d.DeployL1Contracts(ctx)
	d.StartDriverNodes(ctx)
	d.StartProverNodes(ctx)
	d.StartProposerNodes(ctx)
	return nil
}

// Init initializes the network
func (d *Devnet) Init() {
	clientTypes, err := d.t.Sim.ClientTypes()
	if err != nil {
		d.t.Fatalf("failed to retrieve list of client types: %v", err)
	}
	d.clients = Roles(d.t, clientTypes)

	d.L1Genesis, err = getL1Genesis()
	if err != nil {
		d.t.Fatal(err)
	}
	d.L2Genesis = core.TaikoGenesisBlock(d.rollupConf.L2.NetworkID)
	d.L1Vault = NewVault(d.t, d.L1Genesis.Config)
	d.L2Vault = NewVault(d.t, d.L2Genesis.Config)
}

func getL1Genesis() (*core.Genesis, error) {
	g := new(core.Genesis)
	data, err := ioutil.ReadFile("/genesis.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, g); err != nil {
		return nil, err
	}
	return g, nil
}

// StartL1ELNodes starts a eth1 image and add it to the network
func (d *Devnet) StartL1ELNodes(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.L1) == 0 {
		d.t.Fatal("no eth1 client types found")
	}
	opts = append(opts, hivesim.Params{
		envTaikoL1ChainID:      d.rollupConf.L1.ChainID.String(),
		envTaikoL1CliquePeriod: strconv.FormatUint(d.rollupConf.L1.MineInterval, 10),
	})

	for _, c := range d.clients.L1 {
		d.l1Engines = append(d.l1Engines, &ELNode{d.t.StartClient(c.Name, opts...)})
	}
	WaitELNodesUp(ctx, d.t, d.l1Engines, 10*time.Second)
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l1Engines) {
		d.t.Fatalf("only have %d l1 nodes, cannot find %d", len(d.l1Engines), idx)
	}
	return d.l1Engines[idx]
}

func (d *Devnet) StartL2ELNodes(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()
	opts = append(opts, hivesim.Params{
		envTaikoNetworkID: strconv.FormatUint(d.rollupConf.L2.NetworkID, 10),
		envTaikoJWTSecret: d.rollupConf.L2.JWTSecret,
	})
	for i, c := range d.clients.L2 {
		if i > 0 {
			l2 := d.GetL2ELNode(i - 1)
			enodeURL, err := l2.EnodeURL()
			if err != nil {
				d.t.Fatalf("failed to get enode url of the first taiko geth node, error: %w", err)
			}
			opts = append(opts, hivesim.Params{
				envTaikoBootNode: enodeURL,
			})
		}
		d.l2Engines = append(d.l2Engines, &ELNode{d.t.StartClient(c.Name, opts...)})
	}
	WaitELNodesUp(ctx, d.t, d.l2Engines, 10*time.Second)
}

func (d *Devnet) StartDriverNodes(ctx context.Context, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	for i, c := range d.clients.Driver {
		l1 := d.GetL1ELNode(i % len(d.l1Engines))
		l2 := d.GetL2ELNode(i)
		o := append(opts, hivesim.Params{
			envTaikoRole:                            taikoDriver,
			envTaikoL1RPCEndpoint:                   l1.WsRpcEndpoint(),
			envTaikoL2RPCEndpoint:                   l2.WsRpcEndpoint(),
			envTaikoL2EngineEndpoint:                l2.EngineEndpoint(),
			envTaikoL1RollupAddress:                 d.deployments.L1RollupAddress.Hex(),
			envTaikoL2RollupAddress:                 d.deployments.L2RollupAddress.Hex(),
			envTaikoThrowawayBlockBuilderPrivateKey: d.accounts.Throwawayer.PrivateKeyHex,
			"HIVE_CHECK_LIVE_PORT":                  "0",
			envTaikoJWTSecret:                       d.rollupConf.L2.JWTSecret,
		})
		c := d.t.StartClient(c.Name, o...)
		d.drivers = append(d.drivers, &DriverNode{c})
	}
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l2Engines) {
		d.t.Fatalf("only have %d taiko geth nodes, cannot find %d", len(d.l2Engines), idx)
		return nil
	}
	return d.l2Engines[idx]
}

func (d *Devnet) StartProposerNodes(ctx context.Context) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Proposer) == 0 {
		d.t.Fatalf("no taiko proposer client types found")
	}
	for i, c := range d.clients.Proposer {
		l1 := d.GetL1ELNode(i % len(d.l1Engines))
		l2 := d.GetL2ELNode(i % len(d.l2Engines))
		var opts []hivesim.StartOption
		opts = append(opts, hivesim.Params{
			envTaikoRole:                  taikoProposer,
			envTaikoL1RPCEndpoint:         l1.WsRpcEndpoint(),
			envTaikoL2RPCEndpoint:         l2.WsRpcEndpoint(),
			envTaikoL1RollupAddress:       d.deployments.L1RollupAddress.Hex(),
			envTaikoL2RollupAddress:       d.deployments.L2RollupAddress.Hex(),
			envTaikoProposerPrivateKey:    d.accounts.Proposer.PrivateKeyHex,
			envTaikoSuggestedFeeRecipient: d.accounts.SuggestedFeeRecipient.Address.Hex(),
			envTaikoProposeInterval:       d.rollupConf.L2.ProposeInterval.String(),
			"HIVE_CHECK_LIVE_PORT":        "0",
		},
		)
		if d.rollupConf.L2.ProduceInvalidBlocksInterval != 0 {
			opts = append(opts, hivesim.Params{
				envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(d.rollupConf.L2.ProduceInvalidBlocksInterval, 10),
			})
		}
		d.proposers = append(d.proposers, &ProposerNode{d.t.StartClient(c.Name, opts...)})
	}
}

func (d *Devnet) StartProverNodes(ctx context.Context) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Prover) == 0 {
		d.t.Fatalf("no taiko prover client types found")
		return
	}
	for i, c := range d.clients.Prover {
		l1 := d.GetL1ELNode(i % len(d.l1Engines))
		l2 := d.GetL2ELNode(i % len(d.l2Engines))

		if err := d.addWhitelist(ctx, l1.EthClient()); err != nil {
			d.t.Fatalf("add whitelist failed, err=%v", err)
		}
		var opts []hivesim.StartOption
		opts = append(opts, hivesim.Params{
			envTaikoRole:             taikoProver,
			envTaikoL1RPCEndpoint:    l1.WsRpcEndpoint(),
			envTaikoL2RPCEndpoint:    l2.WsRpcEndpoint(),
			envTaikoL1RollupAddress:  d.deployments.L1RollupAddress.Hex(),
			envTaikoL2RollupAddress:  d.deployments.L2RollupAddress.Hex(),
			envTaikoProverPrivateKey: d.accounts.Prover.PrivateKeyHex,
			"HIVE_CHECK_LIVE_PORT":   "0",
		})
		d.provers = append(d.provers, &ProverNode{d.t.StartClient(c.Name, opts...)})
	}

}

func (d *Devnet) addWhitelist(ctx context.Context, cli *ethclient.Client) error {
	taikoL1, err := bindings.NewTaikoL1Client(d.deployments.L1RollupAddress, cli)
	if err != nil {
		return err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(d.accounts.L1Deployer.PrivateKey, d.rollupConf.L1.ChainID)
	if err != nil {
		return err
	}
	opts.GasTipCap = big.NewInt(1500000000)
	tx, err := taikoL1.WhitelistProver(opts, d.accounts.Prover.Address, true)
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
			envTaikoL2ChainID:          d.rollupConf.L2.ChainID.String(),
			"HIVE_CHECK_LIVE_PORT":     "0",
		})

		n := &ContractNode{d.t.StartClient(d.clients.Contract.Name, opts...)}
		result, err := n.Exec("deploy.sh")
		if err != nil || result.ExitCode != 0 {
			d.t.Fatalf("failed to deploy contract on engine node %d(%s), error: %v, result: %v",
				i, e.Container, err, result)
		}
		d.t.Logf("Deploy contracts on %s %s(%s)", e.Type, e.Container, e.IP)
	}
}
