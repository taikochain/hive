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
	env := &TestEnv{
		Context: ctx,
		Conf:    c,
		Clients: clients,
	}
	env.L1Vault = NewVault(t, env.Conf.L1.ChainID)
	env.L2Vault = NewVault(t, env.Conf.L2.ChainID)
	return env
}

func (env *TestEnv) StartSingleNodeNet(t *hivesim.T) {
	env.StartL1L2ProposerDriver(t)
	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	env.Net.Apply(WithProverNode(NewProverNode(t, env, l1, l2)))
}

func (env *TestEnv) StartL1L2ProposerDriver(t *hivesim.T) {
	env.StartL1L2Driver(t)
	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	env.Net.Apply(WithProposerNode(NewProposerNode(t, env, l1, l2)))
}

func (env *TestEnv) StartL1L2Driver(t *hivesim.T) {
	env.StartL1L2(t)
	l1, l2 := env.Net.GetL1ELNode(0), env.Net.GetL2ELNode(0)
	env.Net.Apply(WithDriverNode(NewDriverNode(t, env, l1, l2, false)))
}

func (env *TestEnv) StartL1L2(t *hivesim.T) {
	l2 := NewL2ELNode(t, env, "")
	l1 := NewL1ELNode(t, env)
	deployL1Contracts(t, env, l1, l2)
	taikoL1 := l1.TaikoL1Client(t)
	var err error
	env.L1Constants, err = rpc.GetProtocolConstants(taikoL1, nil)
	require.NoError(t, err)
	opts := []DevOption{
		WithL2Node(l2),
		WithL1Node(l1),
	}
	env.Net = NewDevnet(t, env.Conf, opts...)
}

func (env *TestEnv) GenSomeL1Blocks(t *hivesim.T, cnt uint64) {
	GenSomeBlocks(t, env.Context, env.Net.GetL1ELNode(0), env.L1Vault, cnt)
}

func (env *TestEnv) GenCommitDelayBlocks(t *hivesim.T) {
	cnt := env.L1Constants.CommitDelayConfirmations.Uint64()
	if cnt == 0 {
		return
	}
	n := env.Net.GetL1ELNode(0)
	require.NotNil(t, n)
	GenSomeBlocks(t, env.Context, n, env.L1Vault, cnt)
}

func (env *TestEnv) GenSomeL2Blocks(t *hivesim.T, cnt uint64) {
	n := env.Net.GetL2ELNode(0)
	require.NotNil(t, n)
	GenSomeBlocks(t, env.Context, n, env.L2Vault, cnt)
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

func RunTests(ctx context.Context, t *hivesim.T, params *RunTestsParams) {
	s := semaphore.NewWeighted(params.Concurrency)
	var done int
	doneCh := make(chan struct{})

	for _, test := range params.Tests {
		go func(test *TestSpec) {
			require.NoError(t, s.Acquire(ctx, 1))
			defer s.Release(1)
			env := &TestEnv{
				Context: ctx,
				Net:     params.Devnet,
			}

			require.NoError(t, s.Acquire(ctx, 1))
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
