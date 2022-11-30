package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/taiko"
	"github.com/stretchr/testify/require"
)

// what we need test of our taiko internal
// 1. propose 2048 blocks at once
// 2. propose bad blocks
// 3. driver sync from zero height
// 4. driver sync from some none zero height
func main() {
	suite := hivesim.Suite{
		Name:        "taiko ops",
		Description: "Test propose, sync and other things",
	}

	for _, test := range tests {
		suite.Add(test)
	}

	sim := hivesim.New()
	hivesim.MustRunSuite(sim, suite)
}

var tests = []*hivesim.TestSpec{
	// {Name: "propose 2048 blocks at once", Run: testPropose2048Blocks},
	{Name: "propose bad blocks", Run: testProposeBadBlocks},
	// {Name: "driver sync from zero height", Run: testDriverSyncFromZeroHeight},
	// {Name: "driver sync from some none zero height", Run: testDriverSyncFromNoneZeroHeight},
}

func testPropose2048Blocks(t *hivesim.T) {
	// TODO
}

func testProposeBadBlocks(t *hivesim.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	d := taiko.NewDevnet(t)
	require.NoError(t, taiko.StartDevnetWithSingleInstance(ctx, d, nil))
	l2 := d.GetL2ELNode(0)
	// taiko.WaitBlock(ctx, l2.EthClient(), 1)
	address := d.L2Vault.CreateAccount(context.Background(), l2.EthClient(), big.NewInt(params.Ether))
	t.Logf("address=%v", address)
}

func testDriverSyncFromZeroHeight(t *hivesim.T) {
	// TODO
}

func testDriverSyncFromNoneZeroHeight(t *hivesim.T) {
	// TODO
}
