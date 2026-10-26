package inbound

import (
	"context"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
)

var (
	_ adapter.InboundManager        = (*Manager)(nil)
	_ adapter.DynamicInboundManager = (*Manager)(nil)
)

type Manager struct {
	registry     adapter.InboundRegistry
	endpoint     adapter.EndpointManager
	access       sync.Mutex
	inbounds     []adapter.Inbound
	inboundByTag map[string]adapter.Inbound
	// scope and stage record how far the manager has been started, so
	// inbounds created afterwards can be brought up to the same stage.
	scope   *adapter.Scope
	stage   adapter.StartStage
	started bool
}

func NewManager(registry adapter.InboundRegistry, endpoint adapter.EndpointManager) *Manager {
	return &Manager{
		registry:     registry,
		endpoint:     endpoint,
		inboundByTag: make(map[string]adapter.Inbound),
	}
}

func inboundName(inbound adapter.Inbound) string {
	return "inbound/" + inbound.Type() + "[" + inbound.Tag() + "]"
}

func (m *Manager) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	m.access.Lock()
	inbounds := m.inbounds
	m.scope = scope
	m.access.Unlock()
	for _, inbound := range inbounds {
		err := scope.Start(inboundName(inbound), inbound, stage)
		if err != nil {
			return err
		}
	}
	m.access.Lock()
	m.stage = stage
	m.started = true
	m.access.Unlock()
	return nil
}

func (m *Manager) Inbounds() []adapter.Inbound {
	m.access.Lock()
	defer m.access.Unlock()
	return m.inbounds
}

func (m *Manager) Get(tag string) (adapter.Inbound, bool) {
	m.access.Lock()
	inbound, found := m.inboundByTag[tag]
	m.access.Unlock()
	if found {
		return inbound, true
	}
	return m.endpoint.Get(tag)
}

func (m *Manager) Create(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, outboundType string, options any) error {
	inbound, err := m.registry.Create(ctx, router, logger, tag, outboundType, options)
	if err != nil {
		return err
	}
	m.access.Lock()
	_, loaded := m.inboundByTag[tag]
	if loaded {
		m.access.Unlock()
		return E.New("duplicate inbound tag: ", tag)
	}
	m.inbounds = append(m.inbounds, inbound)
	m.inboundByTag[tag] = inbound
	started, scope, currentStage := m.started, m.scope, m.stage
	m.access.Unlock()
	if !started {
		return nil
	}
	for _, stage := range adapter.ListStartStages {
		if stage > currentStage {
			break
		}
		err = scope.Start(inboundName(inbound), inbound, stage)
		if err != nil {
			_ = m.Remove(tag)
			return err
		}
	}
	return nil
}

func (m *Manager) Remove(tag string) error {
	m.access.Lock()
	inbound, found := m.inboundByTag[tag]
	if !found {
		m.access.Unlock()
		return E.New("inbound not found: ", tag)
	}
	delete(m.inboundByTag, tag)
	for i, it := range m.inbounds {
		if it == inbound {
			m.inbounds = append(m.inbounds[:i:i], m.inbounds[i+1:]...)
			break
		}
	}
	scope := m.scope
	m.access.Unlock()
	if scope == nil {
		return nil
	}
	return scope.Remove(inbound)
}

func (m *Manager) Close() error {
	m.access.Lock()
	inbounds := m.inbounds
	scope := m.scope
	m.started = false
	m.access.Unlock()
	if scope == nil {
		return nil
	}
	var errs []error
	for _, inbound := range inbounds {
		err := scope.Remove(inbound)
		if err != nil {
			errs = append(errs, E.Cause(err, "close ", inboundName(inbound)))
		}
	}
	return E.Errors(errs...)
}
