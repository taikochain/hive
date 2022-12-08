package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
)

var allTests = []*taiko.TestSpec{
	// {Name: "invalid txList", Description: "get invalid txList from L2-engine", Run: genInvalidL2Block},
	// {Name: "L1 reorg", Description: "driver handle L1 re-org", Run: driverHandleL1Reorg},
	// {},
	// {Name: "sync by p2p", Description: "L2 chain head determined by L1, but sync block completes through taiko-geth P2P", Run: syncByP2P},
	// {Name: "propose 2048 blocks at once", Description: "", Run: testPropose2048Blocks},
}

func main() {
	suit := hivesim.Suite{
		Name:        "taiko ops",
		Description: "Test propose, sync and other things",
	}
	suit.Add(&hivesim.TestSpec{
		Name:        "single node net ops",
		Description: "test ops on single node net",
		Run:         launchTest,
	})

	sim := hivesim.New()
	hivesim.MustRun(sim, suit)
}

func launchTest(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// generate the first L2 transaction
	d := taiko.NewDevnet(ctx, t)
	d.L2Vault.CreateAccount(ctx, d.GetL2ELNode(0).EthClient(), big.NewInt(params.Ether))

	t.Run(hivesim.TestSpec{
		Name:        "firstVerifiedL2Block",
		Description: "watch prove event of the first L2 block on L1",
		Run:         firstVerifiedL2Block(t, ctx, d),
	})

	t.Run(hivesim.TestSpec{
		Name:        "sync from L1",
		Description: "completes sync purely from L1 data to generate L2 block",
		Run:         syncAllFromL1(t, ctx, d),
	})
}

// wait the a L2 transaction be proposed and proved as a L2 block.
func firstVerifiedL2Block(t *hivesim.T, ctx context.Context, d *taiko.Devnet) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		blockHash := taiko.GetBlockHashByNumber(ctx, t, d.GetL2ELNode(0).EthClient(), common.Big1, true)
		taiko.WaitProveEvent(ctx, t, d.GetL1ELNode(0), blockHash)
	}
}

func genInvalidL2Block(t *hivesim.T, evn *taiko.TestEnv) {
	// TODO
}

func driverHandleL1Reorg(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}

// Start a new driver and taiko-geth, the driver is connected to L1 that already has a propose block,
// and the driver will synchronize and process the propose event on L1 to let taiko-geth generate a new block.
func syncAllFromL1(t *hivesim.T, ctx context.Context, d *taiko.Devnet) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		l2 := d.AddL2ELNode(ctx, 0, false)
		d.AddDriverNode(ctx, d.GetL1ELNode(0), l2, false)
		taiko.WaitBlock(ctx, t, l2.EthClient(), common.Big1)
	}
}

func syncByP2P(t *hivesim.T, ctx context.Context, d *taiko.Devnet) func(t *hivesim.T) {
	return func(t *hivesim.T) {

	}
}

func testPropose2048Blocks(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}
