package taiko

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/hive/hivesim"
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

// Ethereum node definition
type Node struct {
	*hivesim.Client

	RPC   *rpc.Client
	Eth   *ethclient.Client
	WsRPC *rpc.Client
	WsEth *ethclient.Client
}

func NewNode(name string, t *hivesim.T, opts ...hivesim.StartOption) *Node {
	// create client by hive api
	cli := t.StartClient(name, opts...)
	t.Logf("add node = %s, ip = %s", name, cli.IP)

	n := &Node{Client: cli}

	n.RPC = cli.RPC()
	n.Eth = ethclient.NewClient(n.RPC)

	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	n.WsRPC, _ = rpc.DialWebsocket(ctx, n.WsRpcEndpoint(), "")
	done()

	n.WsEth = ethclient.NewClient(n.WsRPC)
	return n
}

func (n *Node) EnodeURL() (string, error) {
	return n.Client.EnodeURL()
}

func (n *Node) HttpRpcEndpoint() string {
	return fmt.Sprintf("http://%v:%d", n.Client.IP, httpRPCPort)
}

func (n *Node) EngineEndpoint() string {
	return fmt.Sprintf("ws://%v:%d", n.Client.IP, enginePort)
}

func (n *Node) WsRpcEndpoint() string {
	// carried over from older mergenet ws connection problems, idk why clients are different
	switch n.Client.Type {
	case "besu":
		return fmt.Sprintf("ws://%v:%d/ws", n.Client.IP, wsRPCPort)
	case "nethermind":
		return fmt.Sprintf("http://%v:%d/ws", n.Client.IP, wsRPCPort) // upgrade
	default:
		return fmt.Sprintf("ws://%v:%d", n.Client.IP, wsRPCPort)
	}
}

func (n *Node) Exec(command ...string) (*hivesim.ExecInfo, error) {
	return n.Client.Exec(command...)
}
