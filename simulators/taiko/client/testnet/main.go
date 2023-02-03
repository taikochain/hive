package main

import (
	"context"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
	"github.com/taikoxyz/taiko-client/testutils"
)

var tests = []*hivesim.TestSpec{
	{
		Name:        "Generate the first taiko block",
		Description: "Tests related to the first generated taiko block",
		Run:         firstTaikoBlock,
	},
	{
		Name:        "Sync taiko block",
		Description: "Tests related to the synchronization of Taiko blocks",
		Run:         syncTaikoBlock,
	},
	{
		Name:        "Too many pending blocks block the propose process",
		Description: "Too many pending blocks will block further proposes",
		Run:         tooManyPendingBlocks,
	},
	{
		Name:        "Propose invalid transaction list bytes",
		Description: "Commits and proposes an invalid transaction list bytes to TaikoL1 contract.",
		Run:         proposeInvalidTxListBytes,
	},
	{
		Name:        "Propose transaction list including invalid transactions",
		Description: "Commits and proposes a validly encoded transaction list which including an invalid transaction.",
		Run:         proposeTxListIncludingInvalidTx,
	},
	{
		Name:        "Failed to propose by ws because there are too many pending transaction",
		Description: "Total size of pending transactions affects the execution of propose, connected with taiko-geth by ws rpc will fail, when by http will success.",
		Run:         generateLargeTxLists,
	},
}

func main() {
	suit := hivesim.Suite{
		Name:        "taiko-client",
		Description: "Test propose, sync and other things",
	}
	suit.Add(&hivesim.TestSpec{
		Name: "Client launch",
		Description: `This test launches the client and runs the test tool.
		Results from the test tool are reported as individual sub-tests.`,
		Run: func(t *hivesim.T) { runAllTests(t) },
	})
	sim := hivesim.New()
	hivesim.MustRun(sim, suit)
}

func firstTaikoBlock(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	env := taiko.NewTestEnv(ctx, t)
	env.StartSingleNodeNet()
	defer env.StopSingleNodeNet()

	// generate the first L2 transaction
	cli, err := env.Net.GetL2ELNode(0).EthClient()
	require.NoError(t, err)
	env.L2Vault.CreateAccount(ctx, cli, big.NewInt(params.Ether))

	t.Run(hivesim.TestSpec{
		Name: "Generate the first taiko block on L1 chain",
		Run:  GenFirstTaikoBlockOnL1(env),
	})

	t.Run(hivesim.TestSpec{
		Name:        "Verify the first taiko block",
		Description: "Watch prove event of the first taiko block on L1",
		Run:         VerifyFirstTaikoBlockOnL1(env),
	})
}

func GenFirstTaikoBlockOnL1(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		env.GenCommitDelayBlocks(t)
		require.NoError(t, taiko.WaitHeight(env.Context, env.Net.GetL1ELNode(0), taiko.GreaterEqual(1)))
	}
}

// wait the a taiko transaction be proposed and proved as a taiko block.
func VerifyFirstTaikoBlockOnL1(env *taiko.TestEnv) func(*hivesim.T) {
	ctx := env.Context
	return func(t *hivesim.T) {
		l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
		blockHash, err := taiko.GetBlockHashByNumber(ctx, l2, common.Big1, true)
		require.NoError(t, err)
		require.NoError(t, taiko.WaitProveEvent(ctx, l1, blockHash))
		require.NoError(t, taiko.WaitStateChange(l1, func(psv *bindings.ProtocolStateVariables) bool {
			return psv.LatestVerifiedHeight == 1
		}))
	}
}

func syncTaikoBlock(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	env := taiko.NewTestEnv(ctx, t)
	env.StartSingleNodeNet()
	defer env.StopSingleNodeNet()

	blockCnt := uint64(10)
	env.GenSomeL2Blocks(t, blockCnt)

	t.Run(hivesim.TestSpec{
		Name:        "Sync from L1",
		Description: "Completes sync purely from L1 data to generate taiko block",
		Run:         syncAllFromL1(env, blockCnt),
	})
	t.Run(hivesim.TestSpec{
		Name:        "Sync by snap syncing of taiko engine",
		Description: "Taiko chain head determined by L1, but sync block completes through taiko-geth snap mode",
		Run:         syncBySnap(env),
	})
	t.Run(hivesim.TestSpec{
		Name:        "Sync by full syncing of taiko engine",
		Description: "L2 chain head determined by L1, but sync block completes through taiko-geth full mode",
		Run:         syncByFull(env),
	})
	t.Run(hivesim.TestSpec{
		Name:        "Cross-synchronized by p2p syncing of taiko engine and L1",
		Description: "For more complicated synchronization scenarios, first synchronize to latestVerifiedHeight through P2P, and then synchronize through one-by-one, and simulate the driver disconnection in the middle process.",
		Run:         crossSync(env),
	})
}

func syncAllFromL1(env *taiko.TestEnv, height uint64) func(*hivesim.T) {
	return func(t *hivesim.T) {
		l2 := env.NewFullSyncL2ELNode()
		env.NewDriverNode(env.Net.GetL1ELNode(0), l2)
		require.NoError(t, taiko.WaitHeight(env.Context, l2, taiko.GreaterEqual(height)))
	}
}

func syncBySnap(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		// 1. start new L2 engine and driver to sync by p2p
		newL2 := env.NewL2ELNode(taiko.WithBootNode(d.GetL2ENodes(t)))
		env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		// 2. newL2 should sync to LatestVerifiedHeight by snap
		taikoL1, err := d.GetL1ELNode(0).TaikoL1Client()
		require.NoError(t, err)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		heightOfP2PSyncTo := l1State.LatestVerifiedHeight
		require.NoError(t, taiko.WaitHeight(ctx, newL2, taiko.GreaterEqual(heightOfP2PSyncTo)))
	}
}

func syncByFull(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		// 1. start new L2 engine and driver to sync by p2p
		newL2 := env.NewFullSyncL2ELNode(taiko.WithBootNode(d.GetL2ENodes(t)))
		env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		// 2. newL2 should sync to LatestVerifiedHeight by snap
		taikoL1, err := d.GetL1ELNode(0).TaikoL1Client()
		require.NoError(t, err)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		heightOfP2PSyncTo := l1State.LatestVerifiedHeight
		require.NoError(t, taiko.WaitHeight(ctx, newL2, taiko.GreaterEqual(heightOfP2PSyncTo)))
	}
}

func crossSync(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		ctx, d := env.Context, env.Net
		// 1. start new L2 engine and driver to sync by p2p
		newL2 := env.NewFullSyncL2ELNode(taiko.WithBootNode(d.GetL2ENodes(t)))
		newDriver := env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		// 2. newL2 should sync to LatestVerifiedHeight by p2p
		taikoL1, err := d.GetL1ELNode(0).TaikoL1Client()
		require.NoError(t, err)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		heightOfP2PSyncTo := l1State.LatestVerifiedHeight
		require.NoError(t, taiko.WaitHeight(ctx, newL2, taiko.GreaterEqual(heightOfP2PSyncTo)))
		// 3. newDriver should sync one by one from L1
		blockCnt := uint64(10)
		env.GenSomeL2Blocks(t, blockCnt)
		heightOfOneByOne := heightOfP2PSyncTo + blockCnt
		require.NoError(t, taiko.WaitHeight(ctx, newL2, taiko.GreaterEqual(heightOfOneByOne)))
		// 4. stop newDriver, simulate the driver disconnection
		t.Sim.StopClient(t.SuiteID, t.TestID, newDriver.Container)
		// 5. restart newDriver, newL2 should sync to latest header by p2p
		env.GenSomeL2Blocks(t, blockCnt)
		env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		heightOfResume := heightOfOneByOne + blockCnt
		require.NoError(t, taiko.WaitHeight(ctx, newL2, taiko.GreaterEqual(heightOfResume)))
	}
}

// Since there is no prover, state.LatestVerifiedId is always 0,
// so you will get an error when you propose the LibConstants.K_MAX_NUM_BLOCKS block
func tooManyPendingBlocks(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t)
	env.StartL1L2Driver(taiko.WithELNodeType("full"))

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)

	prop, err := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))
	require.NoError(t, err)

	taikoL1, err := l1.TaikoL1Client()
	require.NoError(t, err)
	l2ethCli, err := l2.EthClient()
	require.NoError(t, err)
	for canPropose(t, env, taikoL1) {
		require.NoError(t, env.L2Vault.SendTestTx(ctx, l2ethCli, nil))
		require.NoError(t, prop.ProposeOp(ctx))
		time.Sleep(10 * time.Millisecond)
	}
	// wait error
	require.NoError(t, env.L2Vault.SendTestTx(ctx, l2ethCli, nil))
	err = prop.ProposeOp(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "L1:tooMany"))
}

func canPropose(t *hivesim.T, env *taiko.TestEnv, taikoL1 *bindings.TaikoL1Client) bool {
	l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
	require.NoError(t, err)
	return l1State.NextBlockID < l1State.LatestVerifiedID+env.TaikoConf.MaxNumBlocks.Uint64()
}

// proposeInvalidTxListBytes commits and proposes an invalid transaction list
// bytes to TaikoL1 contract.
func proposeInvalidTxListBytes(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t)
	env.StartL1L2(taiko.WithELNodeType("full"))

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	p, err := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))
	require.NoError(t, err)

	invalidTxListBytes := testutils.RandomBytes(256)
	meta, commitTx, err := p.CommitTxList(
		env.Context,
		invalidTxListBytes,
		uint64(rand.Int63n(env.TaikoConf.BlockMaxGasLimit.Int64())),
		0,
	)
	require.NoError(t, err)
	env.GenCommitDelayBlocks(t)
	require.NoError(t, p.ProposeTxList(env.Context, meta, commitTx, invalidTxListBytes, 1))
	require.NoError(t, taiko.WaitHeight(ctx, l1, taiko.GreaterEqual(1)))
	require.NoError(t, taiko.WaitStateChange(l1, func(psv *bindings.ProtocolStateVariables) bool {
		if psv.NextBlockID == 2 {
			return true
		}
		return false
	}))
}

// proposeTxListIncludingInvalidTx commits and proposes a validly encoded
// transaction list which including an invalid transaction.
func proposeTxListIncludingInvalidTx(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t)
	env.StartL1L2Driver(taiko.WithELNodeType("full"))

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	p, err := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))
	require.NoError(t, err)

	invalidTx := generateInvalidTransaction(env)

	txListBytes, err := rlp.EncodeToBytes(types.Transactions{invalidTx})
	require.NoError(t, err)

	meta, commitTx, err := p.CommitTxList(env.Context, txListBytes, invalidTx.Gas(), 0)
	require.NoError(t, err)

	env.GenCommitDelayBlocks(t)

	require.NoError(t, p.ProposeTxList(env.Context, meta, commitTx, txListBytes, 1))

	require.NoError(t, taiko.WaitHeight(ctx, l1, taiko.GreaterEqual(1)))
	require.NoError(t, taiko.WaitStateChange(l1, func(psv *bindings.ProtocolStateVariables) bool {
		if psv.NextBlockID == 2 {
			return true
		}
		return false
	}))
	l2Eth, err := l2.EthClient()
	require.NoError(t, err)
	pendingNonce, err := l2Eth.PendingNonceAt(context.Background(), env.Conf.L2.Proposer.Address)
	require.NoError(t, err)
	require.NotEqual(t, invalidTx.Nonce(), pendingNonce)
}

// generateInvalidTransaction creates a transaction with an invalid nonce to
// current L2 world state.
func generateInvalidTransaction(env *taiko.TestEnv) *types.Transaction {
	t := env.T
	opts, err := bind.NewKeyedTransactorWithChainID(env.Conf.L2.Proposer.PrivateKey, env.Conf.L2.ChainID)
	require.NoError(t, err)
	l2 := env.Net.GetL2ELNode(0)
	l2Eth, err := l2.EthClient()
	require.NoError(t, err)
	nonce, err := l2Eth.PendingNonceAt(env.Context, env.Conf.L2.Proposer.Address)
	require.NoError(t, err)

	opts.GasLimit = 300000
	opts.NoSend = true
	opts.Nonce = new(big.Int).SetUint64(nonce + 1024)

	taikoL2, err := l2.TaikoL2Client()
	require.NoError(t, err)
	tx, err := taikoL2.Anchor(opts, common.Big0, common.BytesToHash(testutils.RandomBytes(32)))
	require.NoError(t, err)
	return tx
}

func generateLargeTxLists(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t)
	env.StartL1L2(taiko.WithELNodeType("full"))

	l2ethCli, err := env.Net.GetL2ELNode(0).EthClient()
	require.NoError(t, err)
	// generate some transactions whose total size > wsMessageSizeLimit
	// wsMessageSizeLimit is defined in go-ethereum/rpc/websocket.go
	wsMessageSizeLimit := 15 * 1024 * 1024
	dataSize := 15 * 1024
	data := []byte(strings.Repeat("x", dataSize))
	for i := 0; i < wsMessageSizeLimit/dataSize+1; i++ {
		require.NoError(t, env.L2Vault.SendTestTx(ctx, l2ethCli, data))
	}
	// propose by ws will fail
	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	wsProp, err := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))
	require.NoError(t, err)
	err = wsProp.ProposeOp(ctx)
	require.ErrorContains(t, err, "read limit exceeded")

	// propose by http will success
	c := taiko.NewProposerConfig(env, l1, l2)
	c.L2Endpoint = l2.HttpRpcEndpoint()
	httpProp, err := taiko.NewProposer(t, env, c)
	require.NoError(t, err)
	require.NoError(t, httpProp.ProposeOp(ctx))
}

// runAllTests runs the tests against a client instance.
// Most tests simply wait for tx inclusion in a block so we can run many tests concurrently.
func runAllTests(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	taiko.RunTests(t, ctx, &taiko.RunTestsParams{
		Tests:       tests,
		Concurrency: 15,
	})
}
