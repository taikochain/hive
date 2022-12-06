package taiko

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/taikoxyz/taiko-client/bindings"
)

func WaitELNodesUp(ctx context.Context, t *hivesim.T, node *ELNode, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := node.EthClient().ChainID(ctx); err != nil {
		t.Fatalf("engine node %s should be up within %v,err=%v", node.Type, timeout, err)
	}
}

func WaitBlock(ctx context.Context, client *ethclient.Client, n uint64) error {
	for {
		height, err := client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		if height < n {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	return nil
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

func GetL1State(cli *bindings.TaikoL1Client) (*L1State, error) {
	s := new(L1State)
	var err error
	s.GenesisHeight, s.LatestVerifiedHeight, s.LatestVerifiedId, s.NextBlockId, err = cli.GetStateVariables(nil)
	if err != nil {
		return nil, err
	}
	return s, nil
}
