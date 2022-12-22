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

func main() {
	suit := hivesim.Suite{
		Name:        "taiko ops",
		Description: "Test propose, sync and other things",
	}
	suit.Add(&hivesim.TestSpec{
		Name:        "firstL2Block",
		Description: "Relevant tests for the generation of the first L2 block",
		Run:         firstL2Block,
	})
	suit.Add(&hivesim.TestSpec{
		Name:        "sync test",
		Description: "L2 block synchronization related tests",
		Run:         syncL2Block,
	})
	suit.Add(&hivesim.TestSpec{
		Name:        "tooManyPendingBlocks",
		Description: "Too many pending blocks will block further proposes",
		Run:         tooManyPendingBlocks,
	})
	suit.Add(&hivesim.TestSpec{
		Name:        "proposeInvalidTxListBytes",
		Description: "Commits and proposes an invalid transaction list bytes to TaikoL1 contract.",
		Run:         proposeInvalidTxListBytes,
	})
	suit.Add(&hivesim.TestSpec{
		Name:        "proposeTxListIncludingInvalidTx",
		Description: "Commits and proposes a validly encoded transaction list which including an invalid transaction.",
		Run:         proposeTxListIncludingInvalidTx,
	})
	sim := hivesim.New()
	hivesim.MustRun(sim, suit)
}

func firstL2Block(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartSingleNodeNet()

	// generate the first L2 transaction
	env.L2Vault.CreateAccount(ctx, env.Net.GetL2ELNode(0).EthClient(t), big.NewInt(params.Ether))

	t.Run(hivesim.TestSpec{
		Name: "first L1 block",
		Run:  firstL1Block(env),
	})

	t.Run(hivesim.TestSpec{
		Name:        "firstVerifiedL2Block",
		Description: "watch prove event of the first L2 block on L1",
		Run:         firstVerifiedL2Block(env),
	})
}

func firstL1Block(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		env.GenCommitDelayBlocks(t)
		env.WaitHeight(env.Net.GetL1ELNode(0), taiko.GreaterEqual(1))
	}
}

// wait the a L2 transaction be proposed and proved as a L2 block.
func firstVerifiedL2Block(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
		blockHash := env.GetBlockHashByNumber(l2, common.Big1, true)
		env.WaitProveEvent(l1, blockHash)
		env.WaitStateChange(l1, func(psv *bindings.ProtocolStateVariables) bool {
			return psv.LatestVerifiedHeight == 1
		})
	}
}

func syncL2Block(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartSingleNodeNet()

	blockCnt := uint64(10)
	env.GenSomeL2Blocks(t, blockCnt)

	t.Run(hivesim.TestSpec{
		Name:        "sync from L1",
		Description: "completes sync purely from L1 data to generate L2 block",
		Run:         syncAllFromL1(env, blockCnt),
	})
	t.Run(hivesim.TestSpec{
		Name:        "sync by snap",
		Description: "L2 chain head determined by L1, but sync block completes through taiko-geth snap mode",
		Run:         syncBySnap(env),
	})
	t.Run(hivesim.TestSpec{
		Name:        "sync by full",
		Description: "L2 chain head determined by L1, but sync block completes through taiko-geth full mode",
		Run:         syncByFull(env),
	})
	t.Run(hivesim.TestSpec{
		Name:        "sync by p2p and one by one",
		Description: "For more complicated synchronization scenarios, first synchronize to latestVerifiedHeight through P2P, and then synchronize through one-by-one, and simulate the driver disconnection in the middle process.",
		Run:         syncByBothP2PAndOneByOne(env),
	})
}

func syncAllFromL1(env *taiko.TestEnv, height uint64) func(*hivesim.T) {
	return func(t *hivesim.T) {
		l2 := env.NewFullSyncL2ELNode()
		env.NewDriverNode(env.Net.GetL1ELNode(0), l2)
		env.WaitHeight(l2, taiko.GreaterEqual(height))
	}
}

func syncBySnap(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		d := env.Net
		// 1. start new L2 engine and driver to sync by p2p
		newL2 := env.NewL2ELNode(taiko.WithBootNode(d.GetL2ENodes(t)))
		env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		// 2. newL2 should sync to LatestVerifiedHeight by snap
		taikoL1 := d.GetL1ELNode(0).TaikoL1Client(t)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		heightOfP2PSyncTo := l1State.LatestVerifiedHeight
		env.WaitHeight(newL2, taiko.GreaterEqual(heightOfP2PSyncTo))
	}
}

func syncByFull(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		d := env.Net
		// 1. start new L2 engine and driver to sync by p2p
		newL2 := env.NewFullSyncL2ELNode(taiko.WithBootNode(d.GetL2ENodes(t)))
		env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		// 2. newL2 should sync to LatestVerifiedHeight by snap
		taikoL1 := d.GetL1ELNode(0).TaikoL1Client(t)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		heightOfP2PSyncTo := l1State.LatestVerifiedHeight
		env.WaitHeight(newL2, taiko.GreaterEqual(heightOfP2PSyncTo))
	}
}

func syncByBothP2PAndOneByOne(env *taiko.TestEnv) func(*hivesim.T) {
	return func(t *hivesim.T) {
		d := env.Net
		// 1. start new L2 engine and driver to sync by p2p
		newL2 := env.NewFullSyncL2ELNode(taiko.WithBootNode(d.GetL2ENodes(t)))
		newDriver := env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		// 2. newL2 should sync to LatestVerifiedHeight by p2p
		taikoL1 := d.GetL1ELNode(0).TaikoL1Client(t)
		l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		heightOfP2PSyncTo := l1State.LatestVerifiedHeight
		env.WaitHeight(newL2, taiko.GreaterEqual(heightOfP2PSyncTo))
		// 3. newDriver should sync one by one from L1
		blockCnt := uint64(10)
		env.GenSomeL2Blocks(t, blockCnt)
		heightOfOneByOne := heightOfP2PSyncTo + blockCnt
		env.WaitHeight(newL2, taiko.GreaterEqual(heightOfOneByOne))
		// 4. stop newDriver, simulate the driver disconnection
		t.Sim.StopClient(t.SuiteID, t.TestID, newDriver.Container)
		// 5. restart newDriver, newL2 should sync to latest header by p2p
		env.GenSomeL2Blocks(t, blockCnt)
		env.NewDriverNode(d.GetL1ELNode(0), newL2, taiko.WithEnableL2P2P())
		heightOfResume := heightOfOneByOne + blockCnt
		env.WaitHeight(newL2, taiko.GreaterEqual(heightOfResume))
	}
}

// Since there is no prover, state.LatestVerifiedId is always 0,
// so you will get an error when you propose the LibConstants.K_MAX_NUM_BLOCKS block
func tooManyPendingBlocks(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartL1L2Driver(taiko.WithELNodeType("full"))

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)

	prop := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))

	taikoL1 := l1.TaikoL1Client(t)
	for canPropose(t, env, taikoL1) {
		require.NoError(t, env.L2Vault.SendTestTx(ctx, l2.EthClient(t)))
		require.NoError(t, prop.ProposeOp(env.Context))
		time.Sleep(10 * time.Millisecond)
	}
	// wait error
	require.NoError(t, env.L2Vault.SendTestTx(ctx, l2.EthClient(t)))
	err := prop.ProposeOp(env.Context)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "L1:tooMany"))
}

func canPropose(t *hivesim.T, env *taiko.TestEnv, taikoL1 *bindings.TaikoL1Client) bool {
	l1State, err := rpc.GetProtocolStateVariables(taikoL1, nil)
	require.NoError(t, err)
	return l1State.NextBlockID < l1State.LatestVerifiedID+env.L1Constants.MaxNumBlocks.Uint64()
}

// proposeInvalidTxListBytes commits and proposes an invalid transaction list
// bytes to TaikoL1 contract.
func proposeInvalidTxListBytes(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartL1L2(taiko.WithELNodeType("full"))

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	p := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))

	invalidTxListBytes := testutils.RandomBytes(256)
	meta, commitTx, err := p.CommitTxList(
		env.Context,
		invalidTxListBytes,
		uint64(rand.Int63n(env.L1Constants.BlockMaxGasLimit.Int64())),
		0,
	)
	require.NoError(t, err)
	env.GenCommitDelayBlocks(t)
	require.Nil(t, p.ProposeTxList(env.Context, meta, commitTx, invalidTxListBytes, 1))
	env.WaitHeight(l1, taiko.GreaterEqual(1))
	env.WaitStateChange(l1, func(psv *bindings.ProtocolStateVariables) bool {
		if psv.NextBlockID == 2 {
			return true
		}
		return false
	})
}

// proposeTxListIncludingInvalidTx commits and proposes a validly encoded
// transaction list which including an invalid transaction.
func proposeTxListIncludingInvalidTx(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	env := taiko.NewTestEnv(ctx, t, taiko.DefaultConfig)
	env.StartL1L2Driver(taiko.WithELNodeType("full"))

	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	p := taiko.NewProposer(t, env, taiko.NewProposerConfig(env, l1, l2))

	invalidTx := generateInvalidTransaction(env)

	txListBytes, err := rlp.EncodeToBytes(types.Transactions{invalidTx})
	require.NoError(t, err)

	meta, commitTx, err := p.CommitTxList(env.Context, txListBytes, invalidTx.Gas(), 0)
	require.NoError(t, err)

	env.GenCommitDelayBlocks(t)

	require.Nil(t, p.ProposeTxList(env.Context, meta, commitTx, txListBytes, 1))

	env.WaitHeight(l1, taiko.GreaterEqual(1))
	env.WaitStateChange(l1, func(psv *bindings.ProtocolStateVariables) bool {
		if psv.NextBlockID == 2 {
			return true
		}
		return false
	})
	pendingNonce, err := l2.EthClient(t).PendingNonceAt(context.Background(), env.Conf.L2.Proposer.Address)
	require.Nil(t, err)
	require.NotEqual(t, invalidTx.Nonce(), pendingNonce)
}

// generateInvalidTransaction creates a transaction with an invalid nonce to
// current L2 world state.
func generateInvalidTransaction(env *taiko.TestEnv) *types.Transaction {
	t := env.T
	opts, err := bind.NewKeyedTransactorWithChainID(env.Conf.L2.Proposer.PrivateKey, env.Conf.L2.ChainID)
	require.NoError(t, err)
	l2 := env.Net.GetL2ELNode(0)
	nonce, err := l2.EthClient(t).PendingNonceAt(env.Context, env.Conf.L2.Proposer.Address)
	require.NoError(t, err)

	opts.GasLimit = 300000
	opts.NoSend = true
	opts.Nonce = new(big.Int).SetUint64(nonce + 1024)

	taikoL2 := l2.TaikoL2Client(t)
	tx, err := taikoL2.Anchor(opts, common.Big0, common.BytesToHash(testutils.RandomBytes(32)))
	require.NoError(t, err)
	return tx
}
