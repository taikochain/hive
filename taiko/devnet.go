package taiko

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/hive/hivesim"
)

// Devnet is a taiko network with all necessary components, e.g. L1, L2, driver, proposer, prover etc.
type Devnet struct {
	sync.Mutex
	// nodes
	L1Engines []*ELNode
	L2Engines []*ELNode
	drivers   []*Node
	proposers []*Node
	provers   []*Node

	L2Genesis *core.Genesis
}

type DevOption func(*Devnet) *Devnet

func NewDevnet(t *hivesim.T, c *Config, opts ...DevOption) *Devnet {
	d := &Devnet{}
	d.L2Genesis = core.TaikoGenesisBlock(c.L2.NetworkID)
	for _, o := range opts {
		o(d)
	}
	return d
}

func (d *Devnet) GetL1ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.L1Engines) {
		return nil
	}
	return d.L1Engines[idx]
}

func (d *Devnet) GetL2ENodes(t *hivesim.T) string {
	d.Lock()
	defer d.Unlock()
	urls := make([]string, 0)
	for i, n := range d.L2Engines {
		enodeURL, err := n.EnodeURL()
		if err != nil {
			t.Fatalf("failed to get enode url of the %d taiko geth node, error: %v", i, err)
		}
		urls = append(urls, enodeURL)
	}
	return strings.Join(urls, ",")
}

func (d *Devnet) GetL2ELNode(idx int) *ELNode {
	if idx < 0 || idx >= len(d.L2Engines) {
		return nil
	}
	return d.L2Engines[idx]
}
