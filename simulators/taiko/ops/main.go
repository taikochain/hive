package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
)

// what we need test of our taiko internal
// 1. propose 2048 blocks at once
// 2. propose bad blocks
// 3. driver sync from zero height
// 4. driver sync from some none zero height
func main() {
	suit := hivesim.Suite{
		Name:        "taiko ops",
		Description: "Test propose, sync and other things",
	}
	suit.Add(&hivesim.TestSpec{
		Name:        "single node net ops",
		Description: "test ops on single node net",
		Run:         runAllTests(allTests),
	})

	sim := hivesim.New()
	hivesim.MustRun(sim, suit)
}

func runAllTests(tests []*taiko.TestSpec) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		d := taiko.NewDevnet(t, &taiko.NodesConfig{
			L1EngineCnt: 1, L2EngineCnt: 1, ProposerCnt: 1, DriverCnt: 1, ProverCnt: 1})
		require.NoError(t, d.Start(ctx))
		taiko.RunTests(ctx, t, &taiko.RunTestsParams{
			Devnet:      d,
			Tests:       tests,
			Concurrency: 1,
		})
	}
}

var allTests = []*taiko.TestSpec{
	// {Name: "propose 2048 blocks at once", Run: testPropose2048Blocks},
	// {Name: "propose bad blocks", Run: testProposeBadBlocks},
	// {Name: "driver sync from zero height", Run: testDriverSyncFromZeroHeight},
	// {Name: "driver sync from some none zero height", Run: testDriverSyncFromNoneZeroHeight},
	{Name: "example", Run: testExample},
}

func testExample(t *hivesim.T, env *taiko.TestEnv) {
	t.Logf("run example test")
	d := env.Devnet
	l2 := d.GetL2ELNode(0)
	address := d.L2Vault.CreateAccount(env.Ctx(), l2.EthClient(), big.NewInt(params.Ether))
	t.Logf("address=%v", address)
}

func testPropose2048Blocks(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}

func testProposeBadBlocks(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}

func testDriverSyncFromZeroHeight(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}

func testDriverSyncFromNoneZeroHeight(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}
