package inbound

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"

	"github.com/stretchr/testify/require"
)

type testInbound struct {
	tag     string
	stages  []adapter.StartStage
	started atomic.Bool
	closed  atomic.Int32
}

func (i *testInbound) Type() string { return "test" }
func (i *testInbound) Tag() string  { return i.tag }

func (i *testInbound) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	i.stages = append(i.stages, stage)
	if stage == adapter.StartStateStart {
		i.started.Store(true)
		scope.Add(func() error {
			i.closed.Add(1)
			return nil
		})
	}
	return nil
}

type testRegistry struct {
	adapter.InboundRegistry
	created map[string]*testInbound
}

func (r *testRegistry) Create(_ context.Context, _ adapter.Router, _ log.ContextLogger, tag string, _ string, _ any) (adapter.Inbound, error) {
	inbound := &testInbound{tag: tag}
	r.created[tag] = inbound
	return inbound, nil
}

type testEndpointManager struct {
	adapter.EndpointManager
}

func (testEndpointManager) Get(string) (adapter.Endpoint, bool) { return nil, false }

func startManager(t *testing.T, m *Manager, scope *adapter.Scope) {
	for _, stage := range adapter.ListStartStages {
		require.NoError(t, scope.Start("inbound", m, stage))
	}
}

func TestManagerDynamicInbound(t *testing.T) {
	registry := &testRegistry{created: make(map[string]*testInbound)}
	m := NewManager(registry, testEndpointManager{})
	ctx := context.Background()
	logger := log.NewNOPFactory().Logger()
	root := adapter.NewScope(ctx, logger)

	require.NoError(t, m.Create(ctx, nil, logger, "a", "test", nil))
	require.False(t, registry.created["a"].started.Load(), "not started before manager start")
	startManager(t, m, root)
	require.Equal(t, adapter.ListStartStages, registry.created["a"].stages)

	// Created after start: brought up through all stages immediately.
	require.NoError(t, m.Create(ctx, nil, logger, "b", "test", nil))
	b := registry.created["b"]
	require.Equal(t, adapter.ListStartStages, b.stages)
	require.True(t, b.started.Load())

	// Remove releases only that inbound and frees the tag for reuse.
	require.NoError(t, m.Remove("b"))
	require.EqualValues(t, 1, b.closed.Load())
	require.EqualValues(t, 0, registry.created["a"].closed.Load())
	_, found := m.Get("b")
	require.False(t, found)
	require.Error(t, m.Remove("b"))
	require.NoError(t, m.Create(ctx, nil, logger, "b", "test", nil))
	require.Len(t, m.Inbounds(), 2)

	// Close stops all inbounds; closing the box scope afterwards must not
	// close them a second time.
	require.NoError(t, m.Close())
	require.EqualValues(t, 1, registry.created["a"].closed.Load())
	require.EqualValues(t, 1, registry.created["b"].closed.Load())
	require.NoError(t, root.Close())
	require.EqualValues(t, 1, registry.created["a"].closed.Load())
	require.EqualValues(t, 1, registry.created["b"].closed.Load())
	require.EqualValues(t, 1, b.closed.Load())
}
