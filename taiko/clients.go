package taiko

import "github.com/ethereum/hive/hivesim"

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
	L2       []*hivesim.ClientDefinition
	Driver   []*hivesim.ClientDefinition
	Proposer []*hivesim.ClientDefinition
	Prover   []*hivesim.ClientDefinition
	Contract *hivesim.ClientDefinition
}

func Roles(clientDefs []*hivesim.ClientDefinition) *ClientsByRole {
	var out ClientsByRole
	for _, client := range clientDefs {
		if client.HasRole(taikoL1) {
			out.L1 = append(out.L1, client)
		}
		if client.HasRole(taikoDriver) {
			out.Driver = append(out.Driver, client)
		}
		if client.HasRole(taikoGeth) {
			out.L2 = append(out.L2, client)
		}
		if client.HasRole(taikoProposer) {
			out.Proposer = append(out.Proposer, client)
		}
		if client.HasRole(taikoProver) {
			out.Prover = append(out.Prover, client)
		}
		if client.HasRole(taikoProtocol) {
			out.Contract = client
		}
	}
	return &out
}
