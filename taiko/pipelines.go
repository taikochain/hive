package taiko

import (
	"context"
	"strconv"
	"time"

	"github.com/ethereum/hive/hivesim"
)

type PipelineParams struct {
	ProduceInvalidBlocksInterval uint64
}

// StartDevnetWithSingleInstance each component runs only one instance
func StartDevnetWithSingleInstance(ctx context.Context, d *Devnet, params *PipelineParams) error {
	d.Init()
	// start l1 node
	d.AddL1(ctx)
	d.WaitUpL1(ctx, 0, 10*time.Second)
	// deploy l1 contracts
	d.AddProtocol(ctx, 0)
	d.RunDeployL1(ctx)
	// start l2
	d.AddL2(ctx)
	d.WaitUpL2(ctx, 0, 10*time.Second)
	// add components
	if params != nil && params.ProduceInvalidBlocksInterval != 0 {
		d.AddProposer(ctx, 0, 0, hivesim.Params{
			envTaikoProduceInvalidBlocksInterval: strconv.FormatUint(params.ProduceInvalidBlocksInterval, 10),
		})
	} else {
		d.AddProposer(ctx, 0, 0)
	}
	d.AddProver(ctx, 0, 0)
	// init bindings for tests
	d.InitBindingsL1(0)
	d.InitBindingsL2(0)
	return d.WaitL1Block(ctx, 0, 2)
}
