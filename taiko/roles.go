package taiko

import (
	"strings"

	"github.com/ethereum/hive/hivesim"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/taikoxyz/taiko-client/proposer"
	"github.com/taikoxyz/taiko-client/prover"
)

const (
	taikoL1       = "taiko-l1"
	taikoDriver   = "taiko-driver"
	taikoGeth     = "taiko-geth"
	taikoProposer = "taiko-proposer"
	taikoProver   = "taiko-prover"
	taikoProtocol = "taiko-protocol"
)

// ClientsByRole is a collection of ClientDefinitions, grouped by role.
type ClientsByRole struct {
	L1       []*hivesim.ClientDefinition
	L2       *hivesim.ClientDefinition
	Driver   *hivesim.ClientDefinition
	Proposer *hivesim.ClientDefinition
	Prover   *hivesim.ClientDefinition
	Contract *hivesim.ClientDefinition
}

func (r *ClientsByRole) GetL1(isHardhat bool) *hivesim.ClientDefinition {
	if isHardhat {
		for _, d := range r.L1 {
			if strings.Contains(d.Name, "taiko-l1-hardhat") {
				return d
			}
		}
	} else {
		if len(r.L1) > 0 {
			return r.L1[0]
		}
	}
	return nil
}

func Roles(t *hivesim.T) *ClientsByRole {
	clientDefs, err := t.Sim.ClientTypes()
	require.NoError(t, err, "failed to retrieve list of client types: %v", err)
	var out ClientsByRole
	for _, client := range clientDefs {
		if client.HasRole(taikoL1) {
			out.L1 = append(out.L1, client)
		}
		if client.HasRole(taikoDriver) {
			out.Driver = client
		}
		if client.HasRole(taikoGeth) {
			out.L2 = client
		}
		if client.HasRole(taikoProposer) {
			out.Proposer = client
		}
		if client.HasRole(taikoProver) {
			out.Prover = client
		}
		if client.HasRole(taikoProtocol) {
			out.Contract = client
		}
	}
	return &out
}

func NewProposerConfig(env *TestEnv, l1, l2 *ELNode) *proposer.Config {
	return &proposer.Config{
		L1Endpoint:              l1.WsRpcEndpoint(),
		L2Endpoint:              l2.WsRpcEndpoint(),
		TaikoL1Address:          env.Conf.L1.RollupAddress,
		TaikoL2Address:          env.Conf.L2.RollupAddress,
		L1ProposerPrivKey:       env.Conf.L2.Proposer.PrivateKey,
		L2SuggestedFeeRecipient: env.Conf.L2.SuggestedFeeRecipient.Address,
		ProposeInterval:         env.Conf.L2.ProposeInterval,
		ShufflePoolContent:      true,
	}
}

func NewProposer(t *hivesim.T, env *TestEnv, c *proposer.Config) *proposer.Proposer {
	p := new(proposer.Proposer)
	proposer.InitFromConfig(env.Context, p, c)
	return p
}

func NewProverConfig(env *TestEnv) *prover.Config {
	return &prover.Config{
		// TODO
	}
}

func NewProver(t *hivesim.T, env *TestEnv, c *prover.Config) *prover.Prover {
	p := new(prover.Prover)
	prover.InitFromConfig(env.Context, p, c)
	return p
}
