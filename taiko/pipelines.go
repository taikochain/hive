package taiko

import (
	"context"
	"time"
)

// StartTaikoDevnetWithSingleInstance each component runs only one instance
func StartTaikoDevnetWithSingleInstance(ctx context.Context, d *Devnet) error {
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
	d.AddProposer(ctx, 0, 0)
	d.AddProver(ctx, 0, 0)
	// init bindings for tests
	d.InitBindingsL1(0)
	d.InitBindingsL2(0)
	return d.WaitL1Block(ctx, 0, 2)
}
