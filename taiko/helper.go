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
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
)

func (e *TestEnv) WaitELNodesUp(n *ELNode, timeout time.Duration) {
	ctx, t := e.Context, e.T
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := n.EthClient(t).ChainID(ctx); err != nil {
		t.Fatalf("%s should be up within %v,err=%v", n.Type, timeout, err)
	}
}

func (e *TestEnv) WaitHeight(n *ELNode, f func(uint64) bool) {
	ctx, t := e.Context, e.T
	client := n.EthClient(t)
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

func (e *TestEnv) GetBlockHashByNumber(n *ELNode, num *big.Int, needWait bool) common.Hash {
	ctx, t := e.Context, e.T
	if needWait {
		e.WaitHeight(n, GreaterEqual(num.Uint64()))
	}
	cli := n.EthClient(t)
	block, err := cli.BlockByNumber(ctx, num)
	require.NoError(t, err)
	return block.Hash()
}

func (e *TestEnv) WaitReceiptOK(n *ELNode, hash common.Hash) (*types.Receipt, error) {
	return e.WaitReceipt(n, hash, types.ReceiptStatusSuccessful)
}

func (e *TestEnv) WaitReceiptFailed(n *ELNode, hash common.Hash) (*types.Receipt, error) {
	return e.WaitReceipt(n, hash, types.ReceiptStatusFailed)
}

func (e *TestEnv) WaitReceipt(n *ELNode, hash common.Hash, status uint64) (*types.Receipt, error) {
	ctx, t := e.Context, e.T
	client := n.EthClient(t)
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

func (e *TestEnv) SubscribeHeight(n *ELNode, f func(*big.Int) bool) {
	ctx, t := e.Context, e.T
	ch := make(chan *types.Header)
	cli := n.EthClient(t)
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

func (e *TestEnv) WaitProveEvent(n *ELNode, hash common.Hash) {
	ctx, t := e.Context, e.T
	start := uint64(0)
	opt := &bind.WatchOpts{Start: &start, Context: ctx}
	eventCh := make(chan *bindings.TaikoL1ClientBlockProven)
	taikoL1 := n.TaikoL1Client(t)
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

func (e *TestEnv) WaitStateChange(n *ELNode, f func(*bindings.ProtocolStateVariables) bool) {
	t := e.T
	taikoL1 := n.TaikoL1Client(t)
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

func (e *TestEnv) GenSomeBlocks(n *ELNode, v *Vault, cnt uint64) {
	ctx, t := e.Context, e.T
	cli := n.EthClient(t)
	curr, err := cli.BlockNumber(ctx)
	require.NoError(t, err)
	end := curr + cnt
	for curr < end {
		v.CreateAccount(ctx, cli, big.NewInt(params.GWei))
		curr, err = cli.BlockNumber(ctx)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("generate %d L2 blocks", cnt)
}

func GreaterEqual(want uint64) func(uint64) bool {
	return func(get uint64) bool {
		if get >= want {
			return true
		}
		return false
	}
}
