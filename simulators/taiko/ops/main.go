package main

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
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
	suit.Add(&hivesim.TestSpec{
		Name:        "tooManyPendingBlocks",
		Description: "Too many pending blocks will block further proposes",
		Run:         tooManyPendingBlocks,
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
		Name:        "first L1 block",
		Description: "",
		Run:         firstL1Block(t, env),
	})
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

func firstL1Block(t *hivesim.T, env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		taiko.GenCommitDelayBlocks(t, env)
		taiko.WaitHeight(env.Context, t, env.Net.GetL1ELNode(0).EthClient(t), taiko.Greater(common.Big0.Uint64()))
	}
}

// wait the a L2 transaction be proposed and proved as a L2 block.
func firstVerifiedL2Block(t *hivesim.T, env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		blockHash := taiko.GetBlockHashByNumber(ctx, t, d.GetL2ELNode(0).EthClient(t), common.Big1, true)
		taiko.WaitProveEvent(ctx, t, d.GetL1ELNode(0), blockHash)
	}
}

func genInvalidL2Block(t *hivesim.T, evn *taiko.TestEnv) {
	// TODO
}

func l1Reorg(t *hivesim.T, env *taiko.TestEnv) {
	l1 := env.Net.GetL1ELNode(0)
	taikoL1 := l1.TaikoL1Client(t)
	l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
	require.NoError(t, err)
	l1GethCli := l1.GethClient()
	require.NoError(t, l1GethCli.SetHead(env.Context, big.NewInt(int64(l1State.GenesisHeight))))
	l2 := env.Net.GetL2ELNode(0)
	taiko.WaitLatestBlockEqual(env.Context, t, l2.EthClient(t), common.Big0)
}

// Start a new driver and taiko-geth, the driver is connected to L1 that already has a propose block,
// and the driver will synchronize and process the propose event on L1 to let taiko-geth generate a new block.
func syncAllFromL1(t *hivesim.T, env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		l2 := taiko.NewL2ELNode(t, env, "")
		taiko.NewDriverNode(t, env, d.GetL1ELNode(0), l2, false)
		taiko.WaitHeight(ctx, t, l2.EthClient(t), taiko.Greater(common.Big0.Uint64()))
	}
}

func syncByP2P(t *hivesim.T, env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		l2 := d.GetL2ELNode(0).EthClient(t)
		l2LatestHeight, err := l2.BlockNumber(ctx)
		require.NoError(t, err)
		// generate more L2 transactions for test
		cnt := 2
		for i := 0; i < cnt; i++ {
			env.L2Vault.CreateAccount(ctx, l2, big.NewInt(params.Ether))
			taiko.WaitHeight(ctx, t, l2, taiko.Greater(l2LatestHeight+uint64(i)))
		}
		// start new L2 engine and driver to sync by p2p
		newL2 := taiko.NewL2ELNode(t, env, d.GetL2ENodes(t))
		taiko.NewDriverNode(t, env, d.GetL1ELNode(0), newL2, true)

		taikoL1 := d.GetL1ELNode(0).TaikoL1Client(t)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		if l1State.LatestVerifiedHeight > 0 {
			taiko.WaitHeight(ctx, t, newL2.EthClient(t), taiko.Greater(l1State.LatestVerifiedHeight))
		} else {
			t.Logf("sync by p2p, but LatestVerifiedHeight==0")
		}
	}
}

// Since there is no prover, state.LatestVerifiedId is always 0,
// so you will get an error when you propose the LibConstants.K_MAX_NUM_BLOCKS block
func tooManyPendingBlocks(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartL1L2Driver(t)

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)

	prop := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch := make(chan *types.Header)
		sub, err := l2.EthClient(t).SubscribeNewHead(ctx, ch)
		require.NoError(t, err)
		defer sub.Unsubscribe()
		taikoL1 := env.Net.GetL1ELNode(0).TaikoL1Client(t)
		for {
			select {
			case h := <-ch:
				if canPropose(t, env, taikoL1) {
					t.Logf("current block: %v", h.Number)
					require.NoError(t, env.L2Vault.SendTestTx(ctx, l2.EthClient(t)))
					require.NoError(t, prop.ProposeOp(env.Context))
					continue
				}
				return
			case err := <-sub.Err():
				require.NoError(t, err)
			case <-ctx.Done():
				t.Fatalf("program close before test finish")
			}
		}
	}()

	// gen the first l2 block
	require.NoError(t, env.L2Vault.SendTestTx(ctx, l2.EthClient(t)))
	require.NoError(t, prop.ProposeOp(env.Context))

	// wait the pending block up to LibConstants.K_MAX_NUM_BLOCKS
	wg.Wait()

	// wait error
	err := prop.ProposeOp(env.Context)
	require.Error(t, err)
	require.Equal(t, err.Error(), "L1:tooMany")
}

func canPropose(t *hivesim.T, env *taiko.TestEnv, taikoL1 *bindings.TaikoL1Client) bool {
	l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
	require.NoError(t, err)
	return l1State.NextBlockID < l1State.LatestVerifiedID+env.L1Constants.MaxNumBlocks.Uint64()
}
