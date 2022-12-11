package taiko

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
)

// These ports are exposed by the docker containers, and accessible via the docker network that the hive test runs in.
// These are container-ports: they are not exposed to the host,
// and so multiple containers can use the same port.
// Some eth1 client definitions hardcode them, others make them configurable, these should not be changed.

const (
	httpRPCPort = 8545
	wsRPCPort   = 8546
	enginePort  = 8551
)

type ELNode Node

func (e *ELNode) HttpRpcEndpoint() string {
	return fmt.Sprintf("http://%v:%d", e.IP, httpRPCPort)
}

func (e *ELNode) EngineEndpoint() string {
	return fmt.Sprintf("http://%v:%d", e.IP, enginePort)
}

func (e *ELNode) WsRpcEndpoint() string {
	// carried over from older merge net ws connection problems, idk why clients are different
	switch e.Type {
	case "besu":
		return fmt.Sprintf("ws://%v:%d/ws", e.IP, wsRPCPort)
	case "nethermind":
		return fmt.Sprintf("http://%v:%d/ws", e.IP, wsRPCPort) // upgrade
	default:
		return fmt.Sprintf("ws://%v:%d", e.IP, wsRPCPort)
	}
}

func (e *ELNode) EthClient(t *hivesim.T) *ethclient.Client {
	cli, err := ethclient.Dial(e.WsRpcEndpoint())
	require.NoError(t, err)
	return cli
}

func (e *ELNode) L1TaikoClient(t *hivesim.T) *bindings.TaikoL1Client {
	c := e.EthClient(t)
	cli, err := bindings.NewTaikoL1Client(e.TaikoAddr, c)
	require.NoError(t, err)
	return cli
}

func (e *ELNode) L2TaikoClient(t *hivesim.T) *bindings.V1TaikoL2Client {
	c := e.EthClient(t)
	cli, err := bindings.NewV1TaikoL2Client(e.TaikoAddr, c)
	require.NoError(t, err)
	return cli
}
