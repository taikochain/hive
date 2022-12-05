package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
)

var allTests = []*taiko.TestSpec{
	// {Name: "propose 2048 blocks at once", Run: testPropose2048Blocks},
	// {Name: "propose bad blocks", Run: testProposeBadBlocks},
	// {Name: "driver sync from zero height", Run: testDriverSyncFromZeroHeight},
	// {Name: "driver sync from some none zero height", Run: testDriverSyncFromNoneZeroHeight},
	{Name: "generate and prove first l2 Block", Run: testGenProveFirstL2Block},
}

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
		require.NoError(t, d.StartSingleNodeNet(ctx))
		taiko.RunTests(ctx, t, &taiko.RunTestsParams{
			Devnet:      d,
			Tests:       tests,
			Concurrency: 10,
		})
	}
}

func testGenProveFirstL2Block(t *hivesim.T, env *taiko.TestEnv) {
	d := env.DevNet
	l2 := d.GetL2ELNode(0)
	address := d.L2Vault.CreateAccount(env.Context, l2.EthClient(), big.NewInt(params.Ether))
	t.Logf("address=%v", address)
	l1 := d.GetL1ELNode(0)
	taikoL1, err := l1.L1TaikoClient()
	require.Nil(t, err)

	start := uint64(0)
	opt := &bind.WatchOpts{Start: &start, Context: env.Context}
	sink := make(chan *bindings.TaikoL1ClientBlockProven)
	sub, err := taikoL1.WatchBlockProven(opt, sink, []*big.Int{big.NewInt(1)})
	if err != nil {
		t.Fatal("Failed to watch prove event", err)
	}
	for {
		select {
		case err := <-sub.Err():
			t.Fatal("Failed to watch prove event", err)
		case e := <-sink:
			if e.Id.Uint64() == 1 {
				t.Log("all success")
				return
			}
		case <-env.Context.Done():
			t.Log("test is finished before watch proved event")
			return
		}
	}

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
