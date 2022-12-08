package taiko

import (
	"context"
	"math/big"
	"strconv"
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
	c       *Config
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

	L2Genesis *core.Genesis
}

func NewDevnet(ctx context.Context, t *hivesim.T) *Devnet {
	d := &Devnet{t: t, c: DefaultConfig}

	clientTypes, err := d.t.Sim.ClientTypes()
	if err != nil {
		d.t.Fatalf("failed to retrieve list of client types: %v", err)
	}
	d.clients = Roles(d.t, clientTypes)
	d.L2Genesis = core.TaikoGenesisBlock(d.c.L2.NetworkID)
	d.L1Vault = NewVault(d.t, d.c.L1.ChainID)
	d.L2Vault = NewVault(d.t, d.c.L2.ChainID)

	l2 := d.AddL2ELNode(ctx, 0, false)
	l1 := d.AddL1ELNode(ctx, 0, l2)
	d.AddDriverNode(ctx, l1, l2, false)
	d.AddProverNode(ctx, l1, l2)
	d.AddProposerNode(ctx, l1, l2)
	return d
}

// AddL1ELNode starts a eth1 image and add it to the network
func (d *Devnet) AddL1ELNode(ctx context.Context, Idx uint, l2 *ELNode, opts ...hivesim.StartOption) *ELNode {
	if len(d.clients.L1) == 0 {
		d.t.Fatal("no eth1 client types found")
	}
	opts = append(opts, hivesim.Params{
		envTaikoL1ChainID: d.c.L1.ChainID.String(),
		envNetworkID:      strconv.FormatUint(d.c.L1.NetworkID, 10),
		envCliquePeriod:   strconv.FormatUint(d.c.L1.MineInterval, 10),
	})

	c := d.clients.L1[Idx]
	n := &ELNode{d.t.StartClient(c.Name, opts...), d.c.L1.RollupAddress}
	WaitELNodesUp(ctx, d.t, n, 10*time.Second)

	l2GenesisHash := GetBlockHashByNumber(ctx, d.t, l2.EthClient(), common.Big0)
	d.deployL1Contracts(ctx, n, l2GenesisHash)

	d.Lock()
	defer d.Unlock()
	d.l1Engines = append(d.l1Engines, n)

	return n
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l1Engines) {
		d.t.Fatalf("only have %d L1 nodes, cannot find %d", len(d.l1Engines), idx)
	}
	return d.l1Engines[idx]
}

func (d *Devnet) AddL2ELNode(ctx context.Context, clientIdx uint, enableBootNode bool, opts ...hivesim.StartOption) *ELNode {
	opts = append(opts, hivesim.Params{
		envTaikoJWTSecret: d.c.L2.JWTSecret,
		envNodeType:       "full",
		envNetworkID:      strconv.FormatUint(d.c.L2.NetworkID, 10),
		envLogLevel:       "4",
	})
	if enableBootNode {
		opts = append(opts, d.getBootNodeParam())
	}
	c := d.clients.L2[clientIdx]
	n := &ELNode{d.t.StartClient(c.Name, opts...), d.c.L2.RollupAddress}
	WaitELNodesUp(ctx, d.t, n, 10*time.Second)

	d.Lock()
	defer d.Unlock()
	d.l2Engines = append(d.l2Engines, n)
	return n
}

func (d *Devnet) getBootNodeParam() hivesim.StartOption {
	d.Lock()
	defer d.Unlock()

	urls := make([]string, 0)
	for i, n := range d.l2Engines {
		enodeURL, err := n.EnodeURL()
		if err != nil {
			d.t.Fatalf("failed to get enode url of the %d taiko geth node, error: %v", i, err)
			return nil
		}
		urls = append(urls, enodeURL)
	}
	return hivesim.Params{
		envBootNode: strings.Join(urls, ","),
	}
}
func (d *Devnet) AddDriverNode(ctx context.Context, l1, l2 *ELNode, enableP2P bool, opts ...hivesim.StartOption) *DriverNode {
	c := d.clients.Driver[0]
	opts = append(opts, hivesim.Params{
		envTaikoRole:                            "driver",
		envTaikoL1RPCEndpoint:                   l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:                   l2.WsRpcEndpoint(),
		envTaikoL2EngineEndpoint:                l2.EngineEndpoint(),
		envTaikoL1RollupAddress:                 d.c.L1.RollupAddress.Hex(),
		envTaikoL2RollupAddress:                 d.c.L2.RollupAddress.Hex(),
		envTaikoThrowawayBlockBuilderPrivateKey: d.c.L2.Throwawayer.PrivateKeyHex,
		"HIVE_CHECK_LIVE_PORT":                  "0",
		envTaikoJWTSecret:                       d.c.L2.JWTSecret,
	})
	if enableP2P {
		opts = append(opts, hivesim.Params{
			evnTaikoEnableL2P2P: "true",
		})
	}
	n := &DriverNode{d.t.StartClient(c.Name, opts...)}

	d.Lock()
	defer d.Unlock()
	d.drivers = append(d.drivers, n)
	return n
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.l2Engines) {
		d.t.Fatalf("only have %d taiko geth nodes, cannot find %d", len(d.l2Engines), idx)
		return nil
	}
	return d.l2Engines[idx]
}

func (d *Devnet) AddProposerNode(ctx context.Context, l1, l2 *ELNode) *ProposerNode {
	if len(d.clients.Proposer) == 0 {
		d.t.Fatalf("no taiko proposer client types found")
	}
	var opts []hivesim.StartOption
	opts = append(opts, hivesim.Params{
		envTaikoRole:                  "proposer",
		envTaikoL1RPCEndpoint:         l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:         l2.WsRpcEndpoint(),
		envTaikoL1RollupAddress:       d.c.L1.RollupAddress.Hex(),
		envTaikoL2RollupAddress:       d.c.L2.RollupAddress.Hex(),
		envTaikoProposerPrivateKey:    d.c.L2.Proposer.PrivateKeyHex,
		envTaikoSuggestedFeeRecipient: d.c.L2.SuggestedFeeRecipient.Address.Hex(),
		envTaikoProposeInterval:       d.c.L2.ProposeInterval.String(),
		"HIVE_CHECK_LIVE_PORT":        "0",
	},
	)
	if d.c.L2.ProduceInvalidBlocksInterval != 0 {
		opts = append(opts, hivesim.Params{
			envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(d.c.L2.ProduceInvalidBlocksInterval, 10),
		})
	}
	c := d.clients.Proposer[0]
	n := &ProposerNode{d.t.StartClient(c.Name, opts...)}
	d.Lock()
	defer d.Unlock()
	d.proposers = append(d.proposers, n)
	return n

}

func (d *Devnet) AddProverNode(ctx context.Context, l1, l2 *ELNode) *ProverNode {
	if len(d.clients.Prover) == 0 {
		d.t.Fatalf("no taiko prover client types found")
	}
	if err := d.addWhitelist(ctx, l1.EthClient()); err != nil {
		d.t.Fatalf("add whitelist failed, err=%v", err)
	}
	var opts []hivesim.StartOption
	opts = append(opts, hivesim.Params{
		envTaikoRole:             "prover",
		envTaikoL1RPCEndpoint:    l1.WsRpcEndpoint(),
		envTaikoL2RPCEndpoint:    l2.WsRpcEndpoint(),
		envTaikoL1RollupAddress:  d.c.L1.RollupAddress.Hex(),
		envTaikoL2RollupAddress:  d.c.L2.RollupAddress.Hex(),
		envTaikoProverPrivateKey: d.c.L2.Prover.PrivateKeyHex,
		"HIVE_CHECK_LIVE_PORT":   "0",
	})
	c := d.clients.Prover[0]
	n := &ProverNode{d.t.StartClient(c.Name, opts...)}
	d.Lock()
	defer d.Unlock()
	d.provers = append(d.provers, n)
	return n
}

func (d *Devnet) addWhitelist(ctx context.Context, cli *ethclient.Client) error {
	taikoL1, err := bindings.NewTaikoL1Client(d.c.L1.RollupAddress, cli)
	if err != nil {
		return err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(d.c.L1.Deployer.PrivateKey, d.c.L1.ChainID)
	if err != nil {
		return err
	}
	opts.GasTipCap = big.NewInt(1500000000)
	tx, err := taikoL1.WhitelistProver(opts, d.c.L2.Prover.Address, true)
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
	var opts []hivesim.StartOption
	opts = append(opts, hivesim.Params{
		envTaikoPrivateKey:         d.c.L1.Deployer.PrivateKeyHex,
		envTaikoL1DeployerAddress:  d.c.L1.Deployer.Address.Hex(),
		envTaikoL2GenesisBlockHash: l2GenesisHash.Hex(),
		envTaikoL2RollupAddress:    d.c.L2.RollupAddress.Hex(),
		envTaikoMainnetUrl:         l1Node.HttpRpcEndpoint(),
		envTaikoL2ChainID:          d.c.L2.ChainID.String(),
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
