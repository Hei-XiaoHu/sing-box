package vmess

import (
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-vmess"
	"github.com/sagernet/sing/common"
)

// newService builds a vmess service for users. Services are never mutated
// once in use: runtime updates build a new one and swap it in atomically.
func (h *Inbound) newService(users []option.VMessUser) (*vmess.Service[*adapter.InboundUser], error) {
	service := vmess.NewService[*adapter.InboundUser](adapter.NewUpstreamContextHandler(h.newConnectionEx, h.newPacketConnectionEx), h.serviceOptions...)
	err := service.UpdateUsers(adapter.NewInboundUsers(users, func(it option.VMessUser) string {
		return it.Name
	}), common.Map(users, func(it option.VMessUser) string {
		return it.UUID
	}), common.Map(users, func(it option.VMessUser) int {
		return it.AlterId
	}))
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (h *Inbound) startService(scope *adapter.Scope) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	err := h.service.Load().Start()
	if err != nil {
		return err
	}
	h.started = true
	scope.Add(func() error {
		h.userAccess.Lock()
		defer h.userAccess.Unlock()
		h.started = false
		return h.service.Load().Close()
	})
	return nil
}

func (h *Inbound) updateUsersLocked(users []option.VMessUser) error {
	service, err := h.newService(users)
	if err != nil {
		return err
	}
	if h.started {
		err = service.Start()
		if err != nil {
			return err
		}
	}
	oldService := h.service.Swap(service)
	if h.started {
		oldService.Close()
	}
	h.users = users
	return nil
}

func (h *Inbound) UpdateUsers(users []option.VMessUser) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	return h.updateUsersLocked(users)
}

func (h *Inbound) DelUsers(uuids []string) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	toDelete := make(map[string]struct{})
	for _, uuid := range uuids {
		toDelete[uuid] = struct{}{}
	}
	remaining := make([]option.VMessUser, 0, len(h.users))
	for _, user := range h.users {
		if _, found := toDelete[user.UUID]; !found {
			remaining = append(remaining, user)
		}
	}
	return h.updateUsersLocked(remaining)
}
