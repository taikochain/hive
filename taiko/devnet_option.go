package taiko

func WithL1Node(l1 *ELNode) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.L1Engines = append(d.L1Engines, l1)
		return d
	}
}

func WithL2Node(l2 *ELNode) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.L2Engines = append(d.L2Engines, l2)
		return d
	}
}

func WithDriverNode(n *Node) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.drivers = append(d.drivers, n)
		return d
	}
}

func WithProposerNode(n *Node) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.proposers = append(d.proposers, n)
		return d
	}
}

func WithProverNode(n *Node) DevOption {
	return func(d *Devnet) *Devnet {
		d.Lock()
		defer d.Unlock()
		d.provers = append(d.provers, n)
		return d
	}
}
