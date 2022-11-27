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
	L1            []*hivesim.ClientDefinition
	TaikoGeth     []*hivesim.ClientDefinition
	TaikoDriver   []*hivesim.ClientDefinition
	TaikoProposer []*hivesim.ClientDefinition
	TaikoProver   []*hivesim.ClientDefinition
	TaikoProtocol []*hivesim.ClientDefinition
}

func Roles(clientDefs []*hivesim.ClientDefinition) *ClientsByRole {
	var out ClientsByRole
	for _, client := range clientDefs {
		if client.HasRole(taikoL1) {
			out.L1 = append(out.L1, client)
		}
		if client.HasRole(taikoDriver) {
			out.TaikoDriver = append(out.TaikoDriver, client)
		}
		if client.HasRole(taikoGeth) {
			out.TaikoGeth = append(out.TaikoGeth, client)
		}
		if client.HasRole(taikoProposer) {
			out.TaikoProposer = append(out.TaikoProposer, client)
		}
		if client.HasRole(taikoProver) {
			out.TaikoProver = append(out.TaikoProver, client)
		}
		if client.HasRole(taikoProtocol) {
			out.TaikoProtocol = append(out.TaikoProtocol, client)
		}
	}
	return &out
}
