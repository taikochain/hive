package taiko

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
)

func WaitELNodesUp(ctx context.Context, t *hivesim.T, node *ELNode, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := node.EthClient(t).ChainID(ctx); err != nil {
		t.Fatalf("%s should be up within %v,err=%v", node.Type, timeout, err)
	}
}

func WaitHeight(ctx context.Context, t *hivesim.T, client *ethclient.Client, f func(uint64) bool) {
	for {
		height, err := client.BlockNumber(ctx)
		require.NoError(t, err)
		if f(height) {
			break
		}
		time.Sleep(100 * time.Millisecond)
		continue
	}
}

func WaitLatestBlockEqual(ctx context.Context, t *hivesim.T, client *ethclient.Client, n *big.Int) error {
	for {
		height, err := client.BlockNumber(ctx)
		require.NoError(t, err)
		if height == n.Uint64() {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	return nil
}

func GetBlockHashByNumber(ctx context.Context, t *hivesim.T, cli *ethclient.Client, num *big.Int, needWait bool) common.Hash {
	if needWait {
		WaitHeight(ctx, t, cli, Greater(num.Uint64()-1))
	}
	block, err := cli.BlockByNumber(ctx, num)
	require.NoError(t, err)
	return block.Hash()
}

func WaitReceiptOK(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return WaitReceipt(ctx, client, hash, types.ReceiptStatusSuccessful)
}

func WaitReceiptFailed(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return WaitReceipt(ctx, client, hash, types.ReceiptStatusFailed)
}

func WaitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, status uint64) (*types.Receipt, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if errors.Is(err, ethereum.NotFound) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				continue
			}
		}
		if err != nil {
			return nil, err
		}
		if receipt.Status != status {
			return receipt, fmt.Errorf("expected status %d, but got %d", status, receipt.Status)
		}
		return receipt, nil
	}
}

func SubscribeHeight(ctx context.Context, t *hivesim.T, cli *ethclient.Client, f func(*big.Int) bool) {
	ch := make(chan *types.Header)
	sub, err := cli.SubscribeNewHead(ctx, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	for {
		select {
		case h := <-ch:
			if f(h.Number) {
				return
			}
		case err := <-sub.Err():
			require.NoError(t, err)
		case <-ctx.Done():
			t.Fatalf("program close before test finish")
		}
	}
}

func WaitProveEvent(ctx context.Context, t *hivesim.T, l1 *ELNode, hash common.Hash) {
	taikoL1 := l1.TaikoL1Client(t)
	start := uint64(0)
	opt := &bind.WatchOpts{Start: &start, Context: ctx}
	eventCh := make(chan *bindings.TaikoL1ClientBlockProven)
	sub, err := taikoL1.WatchBlockProven(opt, eventCh, nil)
	defer sub.Unsubscribe()
	if err != nil {
		t.Fatal("Failed to watch prove event", err)
	}
	for {
		select {
		case err := <-sub.Err():
			t.Fatal("Failed to watch prove event", err)
		case e := <-eventCh:
			if e.BlockHash == hash {
				return
			}
		case <-ctx.Done():
			t.Log("test is finished before watch proved event")
			return
		}
	}
}

func WaitStateChange(t *hivesim.T, taikoL1 *bindings.TaikoL1Client, f func(*bindings.ProtocolStateVariables) bool) {
	for i := 0; i < 60; i++ {
		s, err := rpc.GetProtocolStateVariables(taikoL1, nil)
		require.NoError(t, err)
		if f(s) {
			break
		}
		time.Sleep(500 * time.Millisecond)
		continue
	}
}

func GenCommitDelayBlocks(t *hivesim.T, env *TestEnv) {
	l1 := env.Net.GetL1ELNode(0)
	curr, err := l1.EthClient(t).BlockNumber(env.Context)
	require.NoError(t, err)
	cnt := int(env.L1Constants.CommitDelayConfirmations.Uint64())
	for i := 0; i < cnt; i++ {
		env.L1Vault.CreateAccount(env.Context, l1.EthClient(t), big.NewInt(params.GWei))
		WaitHeight(env.Context, t, l1.EthClient(t), Greater(curr+uint64(i)))
	}
}

func Greater(want uint64) func(uint64) bool {
	return func(get uint64) bool {
		if get > want {
			return true
		}
		return false
	}
}
