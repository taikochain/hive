package taiko

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/hive/hivesim"
	"github.com/stretchr/testify/require"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
	"golang.org/x/sync/semaphore"
)

// default timeout for RPC calls
var RPCTimeout = 10 * time.Second

// LoggingRoundTrip writes requests and responses to the test log.
type LoggingRoundTrip struct {
	T     *hivesim.T
	Inner http.RoundTripper
}

func (rt *LoggingRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read and log the request body.
	reqBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	rt.T.Logf(">>  %s", bytes.TrimSpace(reqBytes))
	reqCopy := *req
	reqCopy.Body = io.NopCloser(bytes.NewReader(reqBytes))

	// Do the round trip.
	resp, err := rt.Inner.RoundTrip(&reqCopy)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and log the response bytes.
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respCopy := *resp
	respCopy.Body = io.NopCloser(bytes.NewReader(respBytes))
	rt.T.Logf("<<  %s", bytes.TrimSpace(respBytes))
	return &respCopy, nil
}

type TestSpec struct {
	Name        string
	Description string
	Run         func(t *hivesim.T, env *TestEnv)
}

// TestEnv is the environment of a single test.
type TestEnv struct {
	T           *hivesim.T
	Context     context.Context
	Conf        *Config
	Clients     *ClientsByRole
	L1Vault     *Vault
	L2Vault     *Vault
	Net         *Devnet
	L1Constants *bindings.ProtocolConstants
	// This holds most recent context created by the Ctx method.
	// Every time Ctx is called, it creates a new context with the default
	// timeout and cancels the previous one.
	lastCtx    context.Context
	lastCancel context.CancelFunc
}

func NewTestEnv(ctx context.Context, t *hivesim.T, c *Config) *TestEnv {
	clientTypes, err := t.Sim.ClientTypes()
	if err != nil {
		t.Fatalf("failed to retrieve list of client types: %v", err)
	}
	clients := Roles(t, clientTypes)
	e := &TestEnv{
		T:       t,
		Context: ctx,
		Conf:    c,
		Clients: clients,
	}
	e.L1Vault = NewVault(t, c.L1.ChainID)
	e.L2Vault = NewVault(t, c.L2.ChainID)
	return e
}

func (e *TestEnv) StartSingleNodeNet() {
	e.StartL1L2Driver(WithELNodeType("full"))
	l1, l2 := e.Net.GetL1ELNode(0), e.Net.GetL2ELNode(0)
	e.Net.Apply(
		WithProverNode(e.NewProverNode(l1, l2)),
		WithProposerNode(e.NewProposerNode(l1, l2)),
	)
}

func (e *TestEnv) StartL1L2Driver(l2Opts ...NodeOption) {
	e.StartL1L2(l2Opts...)
	l1, l2 := e.Net.GetL1ELNode(0), e.Net.GetL2ELNode(0)
	e.Net.Apply(WithDriverNode(e.NewDriverNode(l1, l2)))
}

func (e *TestEnv) StartL1L2(l2Opts ...NodeOption) {
	l2 := e.NewL2ELNode(l2Opts...)
	l1 := e.NewL1ELNode()
	e.deployL1Contracts(l1, l2)
	taikoL1 := l1.TaikoL1Client(e.T)
	var err error
	e.L1Constants, err = rpc.GetProtocolConstants(taikoL1, nil)
	require.NoError(e.T, err)
	opts := []DevOption{
		WithL2Node(l2),
		WithL1Node(l1),
	}
	e.Net = NewDevnet(e.T, e.Conf, opts...)
}

func (e *TestEnv) GenSomeL1Blocks(t *hivesim.T, cnt uint64) {
	e.GenSomeBlocks(e.Net.GetL1ELNode(0), e.L1Vault, cnt)
}

func (e *TestEnv) GenCommitDelayBlocks(t *hivesim.T) {
	cnt := e.L1Constants.CommitDelayConfirmations.Uint64()
	if cnt == 0 {
		return
	}
	n := e.Net.GetL1ELNode(0)
	require.NotNil(t, n)
	e.GenSomeBlocks(n, e.L1Vault, cnt)
}

func (e *TestEnv) GenSomeL2Blocks(t *hivesim.T, cnt uint64) {
	n := e.Net.GetL2ELNode(0)
	require.NotNil(t, n)
	e.GenSomeBlocks(n, e.L2Vault, cnt)
}

// Ctx returns a context with the default timeout.
// For subsequent calls to Ctx, it also cancels the previous context.
func (t *TestEnv) Ctx() context.Context {
	return t.TimeoutCtx(RPCTimeout)
}

func (t *TestEnv) TimeoutCtx(timeout time.Duration) context.Context {
	if t.lastCtx != nil {
		t.lastCancel()
	}
	t.lastCtx, t.lastCancel = context.WithTimeout(t.Context, timeout)
	return t.lastCtx
}

type RunTestsParams struct {
	Devnet      *Devnet
	Tests       []*TestSpec
	Concurrency int64
}

func RunTests(env *TestEnv, params *RunTestsParams) {
	s := semaphore.NewWeighted(params.Concurrency)
	var done int
	doneCh := make(chan struct{})

	t, ctx := env.T, env.Context

	for _, test := range params.Tests {
		go func(test *TestSpec) {
			require.NoError(t, s.Acquire(ctx, 1))
			defer s.Release(1)
			t.Run(hivesim.TestSpec{
				Name:        test.Name,
				Description: test.Description,
				Run: func(t *hivesim.T) {
					test.Run(t, env)
					if env.lastCtx != nil {
						env.lastCancel()
					}
				},
			})
			doneCh <- struct{}{}
		}(test)
	}

	for done < len(params.Tests) {
		select {
		case <-doneCh:
			done++
		case <-ctx.Done():
			return
		}
	}
}
