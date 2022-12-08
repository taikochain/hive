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
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
)

func WaitELNodesUp(ctx context.Context, t *hivesim.T, node *ELNode, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := node.EthClient().ChainID(ctx); err != nil {
		t.Fatalf("engine node %s should be up within %v,err=%v", node.Type, timeout, err)
	}
}

func WaitBlock(ctx context.Context, t *hivesim.T, client *ethclient.Client, n *big.Int) error {
	for {
		height, err := client.BlockNumber(ctx)
		require.NoError(t, err)
		if height < n.Uint64() {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	return nil
}

func GetBlockHashByNumber(ctx context.Context, t *hivesim.T, cli *ethclient.Client, num *big.Int, needWait bool) common.Hash {
	if needWait {
		WaitBlock(ctx, t, cli, num)
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

type L1State struct {
	GenesisHeight        uint64
	LatestVerifiedHeight uint64
	LatestVerifiedId     uint64
	NextBlockId          uint64
}

func GetL1State(t *hivesim.T, cli *bindings.TaikoL1Client) *L1State {
	s := new(L1State)
	var err error
	s.GenesisHeight, s.LatestVerifiedHeight, s.LatestVerifiedId, s.NextBlockId, err = cli.GetStateVariables(nil)
	require.NoError(t, err)
	return s
}

func WaitNewHead(ctx context.Context, t *hivesim.T, cli *ethclient.Client, wantHeight *big.Int) {
	ch := make(chan *types.Header)
	sub, err := cli.SubscribeNewHead(ctx, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	for {
		select {
		case h := <-ch:
			if h.Number.Uint64() >= wantHeight.Uint64() {
				return
			}
		case err := <-sub.Err():
			require.NoError(t, err)
		case <-ctx.Done():
			t.Fatalf("program close before test finish")
		}
	}
}

func WaitProveEvent(ctx context.Context, t *hivesim.T, l1 *ELNode, blockHash common.Hash) {
	taikoL1 := l1.L1TaikoClient(t)
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
			if e.BlockHash == blockHash {
				return
			}
		case <-ctx.Done():
			t.Log("test is finished before watch proved event")
			return
		}
	}
}

func WaitStateChange(t *hivesim.T, l1 *ELNode, address common.Address, f func(*L1State) bool) {
	taikoL1, err := bindings.NewTaikoL1Client(address, l1.EthClient())
	require.NoError(t, err)
	for i := 0; i < 60; i++ {
		s := GetL1State(t, taikoL1)
		t.Logf("L1 rollup state=%+v", s)
		if f(s) {
			break
		}
		time.Sleep(time.Second)
		continue
	}
}
