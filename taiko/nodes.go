package taiko

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/taikoxyz/taiko-client/bindings"
)

// These ports are exposed on the docker containers, and accessible via the docker network that the hive test runs in.
// These are container-ports: they are not exposed to the host,
// and so multiple containers can use the same port.
// Some eth1 client definitions hardcode them, others make them configurable, these should not be changed.
const (
	httpRPCPort = 8545
	wsRPCPort   = 8546
	enginePort  = 8551
)

// execution layer node definition
type ELNode struct {
	*hivesim.Client
	TaikoAddr common.Address
}

func (e *ELNode) HttpRpcEndpoint() string {
	return fmt.Sprintf("http://%v:%d", e.IP, httpRPCPort)
}

func (e *ELNode) EngineEndpoint() string {
	return fmt.Sprintf("http://%v:%d", e.IP, enginePort)
}

func (e *ELNode) WsRpcEndpoint() string {
	// carried over from older merge net ws connection problems, idk why clients are different
	switch e.Client.Type {
	case "besu":
		return fmt.Sprintf("ws://%v:%d/ws", e.IP, wsRPCPort)
	case "nethermind":
		return fmt.Sprintf("http://%v:%d/ws", e.IP, wsRPCPort) // upgrade
	default:
		return fmt.Sprintf("ws://%v:%d", e.IP, wsRPCPort)
	}
}

func (e *ELNode) RawRpcClient() (*ethclient.Client, error) {
	return ethclient.Dial(e.WsRpcEndpoint())
}

func (e *ELNode) EthClient() *ethclient.Client {
	return ethclient.NewClient(e.RPC())
}


func (e *ELNode) L1TaikoClient() (*bindings.TaikoL1Client, error) {
	c, err := e.RawRpcClient()
	if err != nil {
		return nil, err
	}
	return bindings.NewTaikoL1Client(e.TaikoAddr, c)
}

func (e *ELNode) L2TaikoClient() (*bindings.V1TaikoL2Client, error) {
	c, err := e.RawRpcClient()
	if err != nil {
		return nil, err
	}
	return bindings.NewV1TaikoL2Client(e.TaikoAddr, c)
}

type DriverNode struct {
	*hivesim.Client
}

type ProposerNode struct {
	*hivesim.Client
}

type ProverNode struct {
	*hivesim.Client
}

type ContractNode struct {
	*hivesim.Client
}
