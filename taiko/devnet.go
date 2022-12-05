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

	accounts *TestAccounts // all accounts will be used in the network

	L1Genesis *core.Genesis
	L2Genesis *core.Genesis

	deployConf *DeployConfig // contracts deployment info
	rollupConf *RollupConfig
	nodesConf  *NodesConfig
}

func NewDevnet(t *hivesim.T, conf *NodesConfig) *Devnet {
	d := &Devnet{
		t:          t,
		nodesConf:  conf,
		deployConf: DefaultDeployments(t),
		rollupConf: DefaultRollupConfig,
		accounts:   DefaultAccounts(t),
	}
	return d
}

func (d *Devnet) StartSingleNodeNet(ctx context.Context) error {
	d.Init()
	d.AddL1ELNode(ctx, 0)
	d.AddL2ELNode(ctx, 0)
	d.AddDriverNode(ctx, d.l1Engines[0], d.l2Engines[0])
	d.AddProverNode(ctx, d.l1Engines[0], d.l2Engines[0])
	d.AddProposerNode(ctx, d.l1Engines[0], d.l2Engines[0])
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

// AddL1ELNode starts a eth1 image and add it to the network
func (d *Devnet) AddL1ELNode(ctx context.Context, Idx uint, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.L1) == 0 {
		d.t.Fatal("no eth1 client types found")
	}
	opts = append(opts, hivesim.Params{
		envTaikoL1ChainID:      d.rollupConf.L1.ChainID.String(),
		envTaikoL1CliquePeriod: strconv.FormatUint(d.rollupConf.L1.MineInterval, 10),
	})

	c := d.clients.L1[Idx]
	n := &ELNode{d.t.StartClient(c.Name, opts...), d.deployConf.L1.RollupAddress}
	WaitELNodesUp(ctx, d.t, n, 10*time.Second)
	d.l1Engines = append(d.l1Engines, n)
	d.deployL1Contracts(ctx, n)
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l1Engines) {
		d.t.Fatalf("only have %d l1 nodes, cannot find %d", len(d.l1Engines), idx)
	}
	return d.l1Engines[idx]
}

func (d *Devnet) AddL2ELNode(ctx context.Context, clientIdx uint, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()
	opts = append(opts, hivesim.Params{
		envTaikoNetworkID: strconv.FormatUint(d.rollupConf.L2.NetworkID, 10),
		envTaikoJWTSecret: d.rollupConf.L2.JWTSecret,
	})
	for _, n := range d.l2Engines {
		enodeURL, err := n.EnodeURL()
		if err != nil {
			d.t.Fatalf("failed to get enode url of the first taiko geth node, error: %w", err)
		}
		opts = append(opts, hivesim.Params{
			envTaikoBootNode: enodeURL,
		})
	}
	c := d.clients.L2[clientIdx]
	n := &ELNode{d.t.StartClient(c.Name, opts...), d.deployConf.L2.RollupAddress}
	WaitELNodesUp(ctx, d.t, n, 10*time.Second)
	d.l2Engines = append(d.l2Engines, n)
}

func (d *Devnet) AddDriverNode(ctx context.Context, l1, l2 *ELNode, opts ...hivesim.StartOption) {
	d.Lock()
	defer d.Unlock()

	if len(d.l2Engines)-len(d.drivers) != 1 {
		d.t.Fatalf("l2 engines number must equals driver number")
	}
	c := d.clients.Driver[0]
	opts = append(opts, hivesim.Params{
		envTaikoRole:                            taikoDriver,
		envTaikoL1RPCEndpoint:                   l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:                   l2.WsRpcEndpoint(),
		envTaikoL2EngineEndpoint:                l2.EngineEndpoint(),
		envTaikoL1RollupAddress:                 d.deployConf.L1.RollupAddress.Hex(),
		envTaikoL2RollupAddress:                 d.deployConf.L2.RollupAddress.Hex(),
		envTaikoThrowawayBlockBuilderPrivateKey: d.deployConf.L2.Throwawayer.PrivateKeyHex,
		"HIVE_CHECK_LIVE_PORT":                  "0",
		envTaikoJWTSecret:                       d.rollupConf.L2.JWTSecret,
	})
	d.drivers = append(d.drivers, &DriverNode{d.t.StartClient(c.Name, opts...)})
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l2Engines) {
		d.t.Fatalf("only have %d taiko geth nodes, cannot find %d", len(d.l2Engines), idx)
		return nil
	}
	return d.l2Engines[idx]
}

func (d *Devnet) AddProposerNode(ctx context.Context, l1, l2 *ELNode) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Proposer) == 0 {
		d.t.Fatalf("no taiko proposer client types found")
	}
	var opts []hivesim.StartOption
	opts = append(opts, hivesim.Params{
		envTaikoRole:                  taikoProposer,
		envTaikoL1RPCEndpoint:         l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:         l2.WsRpcEndpoint(),
		envTaikoL1RollupAddress:       d.deployConf.L1.RollupAddress.Hex(),
		envTaikoL2RollupAddress:       d.deployConf.L2.RollupAddress.Hex(),
		envTaikoProposerPrivateKey:    d.deployConf.L2.Proposer.PrivateKeyHex,
		envTaikoSuggestedFeeRecipient: d.deployConf.L2.SuggestedFeeRecipient.Address.Hex(),
		envTaikoProposeInterval:       d.rollupConf.L2.ProposeInterval.String(),
		"HIVE_CHECK_LIVE_PORT":        "0",
	},
	)
	if d.rollupConf.L2.ProduceInvalidBlocksInterval != 0 {
		opts = append(opts, hivesim.Params{
			envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(d.rollupConf.L2.ProduceInvalidBlocksInterval, 10),
		})
	}
	c := d.clients.Proposer[0]
	d.proposers = append(d.proposers, &ProposerNode{d.t.StartClient(c.Name, opts...)})
}

func (d *Devnet) AddProverNode(ctx context.Context, l1, l2 *ELNode) {
	d.Lock()
	defer d.Unlock()

	if len(d.clients.Prover) == 0 {
		d.t.Fatalf("no taiko prover client types found")
		return
	}
	if err := d.addWhitelist(ctx, l1.EthClient()); err != nil {
		d.t.Fatalf("add whitelist failed, err=%v", err)
	}
	var opts []hivesim.StartOption
	opts = append(opts, hivesim.Params{
		envTaikoRole:             taikoProver,
		envTaikoL1RPCEndpoint:    l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:    l2.WsRpcEndpoint(),
		envTaikoL1RollupAddress:  d.deployConf.L1.RollupAddress.Hex(),
		envTaikoL2RollupAddress:  d.deployConf.L2.RollupAddress.Hex(),
		envTaikoProverPrivateKey: d.deployConf.L2.Prover.PrivateKeyHex,
		"HIVE_CHECK_LIVE_PORT":   "0",
	})
	c := d.clients.Prover[0]
	d.provers = append(d.provers, &ProverNode{d.t.StartClient(c.Name, opts...)})
}

func (d *Devnet) addWhitelist(ctx context.Context, cli *ethclient.Client) error {
	taikoL1, err := bindings.NewTaikoL1Client(d.deployConf.L1.RollupAddress, cli)
	if err != nil {
		return err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(d.deployConf.L1.Deployer.PrivateKey, d.rollupConf.L1.ChainID)
	if err != nil {
		return err
	}
	opts.GasTipCap = big.NewInt(1500000000)
	tx, err := taikoL1.WhitelistProver(opts, d.deployConf.L2.Prover.Address, true)
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
func (d *Devnet) deployL1Contracts(ctx context.Context, l1Node *ELNode) {
	d.Lock()
	defer d.Unlock()

	if d.clients.Contract == nil {
		d.t.Fatalf("no taiko protocol client types found")
	}
	var opts []hivesim.StartOption
	opts = append(opts, hivesim.Params{
		envTaikoPrivateKey:         d.deployConf.L1.Deployer.PrivateKeyHex,
		envTaikoL1DeployerAddress:  d.deployConf.L1.Deployer.Address.Hex(),
		envTaikoL2GenesisBlockHash: d.deployConf.L2.GenesisBlockHash.Hex(),
		envTaikoL2RollupAddress:    d.deployConf.L2.RollupAddress.Hex(),
		envTaikoMainnetUrl:         l1Node.HttpRpcEndpoint(),
		envTaikoL2ChainID:          d.rollupConf.L2.ChainID.String(),
		"HIVE_CHECK_LIVE_PORT":     "0",
	})
	n := &ContractNode{d.t.StartClient(d.clients.Contract.Name, opts...)}
	result, err := n.Exec("deploy.sh")
	if err != nil || result.ExitCode != 0 {
		d.t.Fatalf("failed to deploy contract on engine node %s, error: %v, result: %v",
			l1Node.Container, err, result)
	}
	d.t.Logf("Deploy contracts on %s %s(%s)", l1Node.Type, l1Node.Container, l1Node.IP)

}
