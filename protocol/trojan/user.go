package trojan

import (
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/transport/trojan"
	"github.com/sagernet/sing/common"
)

// newService builds a trojan service for users. Services are never mutated
// once in use: runtime updates build a new one and swap it in atomically.
func (h *Inbound) newService(users []option.TrojanUser) (*trojan.Service[*adapter.InboundUser], error) {
	service := trojan.NewService[*adapter.InboundUser](adapter.NewUpstreamContextHandler(h.newConnection, h.newPacketConnection), h.fallbackHandler, h.logger)
	err := service.UpdateUsers(adapter.NewInboundUsers(users, func(it option.TrojanUser) string {
		return it.Name
	}), common.Map(users, func(it option.TrojanUser) string {
		return it.Password
	}))
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (h *Inbound) UpdateUsers(users []option.TrojanUser) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	service, err := h.newService(users)
	if err != nil {
		return err
	}
	h.service.Store(service)
	h.users = users
	return nil
}
