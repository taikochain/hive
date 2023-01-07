package taiko

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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

type ELNode struct {
	*Node
	deploy      *deployResult
	genesisHash common.Hash
}

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

func (e *ELNode) EthClient() (*ethclient.Client, error) {
	cli, err := ethclient.Dial(e.WsRpcEndpoint())
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (e *ELNode) TaikoL1Client() (*bindings.TaikoL1Client, error) {
	c, err := e.EthClient()
	if err != nil {
		return nil, err
	}
	cli, err := bindings.NewTaikoL1Client(e.deploy.rollupAddress, c)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (e *ELNode) TaikoL2Client() (*bindings.TaikoL2Client, error) {
	c, err := e.EthClient()
	if err != nil {
		return nil, err
	}
	cli, err := bindings.NewTaikoL2Client(e.deploy.rollupAddress, c)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (e *ELNode) GethClient() *gethclient.Client {
	return gethclient.New(e.RPC())
}
