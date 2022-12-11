package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
)

func main() {
	suit := hivesim.Suite{
		Name:        "taiko ops",
		Description: "Test propose, sync and other things",
	}
	suit.Add(&hivesim.TestSpec{
		Name:        "single node net ops",
		Description: "test ops on single node net",
		Run:         singleNodeTest,
	})

	sim := hivesim.New()
	hivesim.MustRun(sim, suit)
}

func singleNodeTest(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartSingleNodeNet(t)

	// generate the first L2 transaction
	env.L2Vault.CreateAccount(ctx, env.Net.GetL2ELNode(0).EthClient(t), big.NewInt(params.Ether))

	t.Run(hivesim.TestSpec{
		Name:        "firstVerifiedL2Block",
		Description: "watch prove event of the first L2 block on L1",
		Run:         firstVerifiedL2Block(t, env),
	})

	t.Run(hivesim.TestSpec{
		Name:        "sync from L1",
		Description: "completes sync purely from L1 data to generate L2 block",
		Run:         syncAllFromL1(t, env),
	})

	t.Run(hivesim.TestSpec{
		Name:        "sync by p2p",
		Description: "L2 chain head determined by L1, but sync block completes through taiko-geth P2P",
		Run:         syncByP2P(t, env),
	})
}

// wait the a L2 transaction be proposed and proved as a L2 block.
func firstVerifiedL2Block(t *hivesim.T, env *taiko.TestEnv) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		blockHash := taiko.GetBlockHashByNumber(ctx, t, d.GetL2ELNode(0).EthClient(t), common.Big1, true)
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
func syncAllFromL1(t *hivesim.T, env *taiko.TestEnv) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		l2 := taiko.NewL2ELNode(t, env, "")
		taiko.NewDriverNode(t, env, d.GetL1ELNode(0), l2, false)
		taiko.WaitBlock(ctx, t, l2.EthClient(t), common.Big1)
	}
}

func syncByP2P(t *hivesim.T, env *taiko.TestEnv) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		l2LatestHeight, err := d.GetL2ELNode(0).EthClient(t).BlockNumber(ctx)
		require.NoError(t, err)
		// generate the second L2 transaction
		cnt := 2
		for i := 0; i < cnt; i++ {
			env.L2Vault.CreateAccount(ctx, d.GetL2ELNode(0).EthClient(t), big.NewInt(params.Ether))
		}
		// wait the L1 state change as expected
		taiko.WaitBlock(ctx, t, d.GetL1ELNode(0).EthClient(t), big.NewInt(int64(l2LatestHeight)+int64(cnt)))
		l2 := taiko.NewL2ELNode(t, env, d.GetL2ENodes(t))
		taiko.NewDriverNode(t, env, d.GetL1ELNode(0), l2, true)
		taiko.WaitBlock(ctx, t, l2.EthClient(t), common.Big2)
	}
}

func proposeTooManyBlocks(t *hivesim.T) {
	// TODO
}
