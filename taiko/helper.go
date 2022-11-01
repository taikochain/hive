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
)

func WaitUp(ctx context.Context, t *hivesim.T, cli *ethclient.Client, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := cli.ChainID(ctx)
	if err != nil {
		t.Fatalf("cannot wait for node: %w", err)
	}
}

func WaitBlock(ctx context.Context, cli *ethclient.Client, atLeastHight uint64) error {
	for {
		height, err := cli.BlockNumber(ctx)
		if err != nil {
			return err
		}
		if height < atLeastHight {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	return nil
}

func WaitReceipt(ctx context.Context, cli *ethclient.Client, hash common.Hash, status uint64) (*types.Receipt, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		receipt, err := cli.TransactionReceipt(ctx, hash)
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

func WaitReceiptOK(ctx context.Context, cli *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return WaitReceipt(ctx, cli, hash, types.ReceiptStatusSuccessful)
}

func WaitReceiptFail(ctx context.Context, cli *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return WaitReceipt(ctx, cli, hash, types.ReceiptStatusFailed)
}
