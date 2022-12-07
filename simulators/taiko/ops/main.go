package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
)

var allTests = []*taiko.TestSpec{
	{Name: "first L2", Description: "generate first verified L2 block", Run: genFirstVerifiedL2Block},
	{Name: "invalid txList", Description: "get invalid txList from l2-engine", Run: genInvalidL2Block},
	{Name: "L1 reorg", Description: "driver handle L1 re-org", Run: driverHandleL1Reorg},
	{Name: "sync from L1", Description: "completes sync purely from L1 data to generate l2 block", Run: syncAllFromL1},
	{Name: "sync by p2p", Description: "l2 chain head determined by L1, but sync block completes through taiko-geth P2P", Run: syncByP2P},
	{Name: "propose 2048 blocks at once", Description: "", Run: testPropose2048Blocks},
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
		d := taiko.NewDevnet(ctx, t)
		taiko.RunTests(ctx, t, &taiko.RunTestsParams{
			Devnet:      d,
			Tests:       tests,
			Concurrency: 10,
		})
	}
}

func genFirstVerifiedL2Block(t *hivesim.T, env *taiko.TestEnv) {
	d := env.DevNet
	l1, l2 := d.GetL1ELNode(0), d.GetL2ELNode(0)
	d.L2Vault.CreateAccount(env.Context, l2.EthClient(), big.NewInt(params.Ether))
	taikoL1, err := l1.L1TaikoClient()
	require.Nil(t, err)

	start := uint64(0)
	opt := &bind.WatchOpts{Start: &start, Context: env.Context}
	eventCh := make(chan *bindings.TaikoL1ClientBlockProven)
	sub, err := taikoL1.WatchBlockProven(opt, eventCh, []*big.Int{big.NewInt(1)})
	defer sub.Unsubscribe()
	if err != nil {
		t.Fatal("Failed to watch prove event", err)
	}
	for {
		select {
		case err := <-sub.Err():
			t.Fatal("Failed to watch prove event", err)
		case e := <-eventCh:
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

func genInvalidL2Block(t *hivesim.T, evn *taiko.TestEnv) {
	// TODO
}

func driverHandleL1Reorg(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}

// Start a new driver and taiko-geth, the driver is connected to L1 that already has a propose block,
// and the driver will synchronize and process the propose event on L1 to let taiko-geth generate a new block.
func syncAllFromL1(t *hivesim.T, env *taiko.TestEnv) {
	d := env.DevNet
	l2 := d.AddL2ELNode(env.Context, 0)
	d.AddDriverNode(env.Context, d.GetL1ELNode(0), l2)

	ch := make(chan *types.Header)
	cli, err := l2.RawRpcClient()
	require.NoError(t, err)
	sub, err := cli.SubscribeNewHead(env.Context, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()
	for {
		for {
			select {
			case h := <-ch:
				if h.Number.Uint64() > 0 {
					return
				}
				return
			case err := <-sub.Err():
				require.NoError(t, err)
			case <-env.Context.Done():
				t.Fatalf("program close before test finish")
			}
		}
	}

}

func syncByP2P(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}
func testPropose2048Blocks(t *hivesim.T, env *taiko.TestEnv) {
	// TODO
}
