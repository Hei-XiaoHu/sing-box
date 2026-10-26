package vless

import (
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-vmess/vless"
	"github.com/sagernet/sing/common"
)

// newService builds a vless service for users. Services are never mutated
// once in use: runtime updates build a new one and swap it in atomically.
func (h *Inbound) newService(users []option.VLESSUser) *vless.Service[*adapter.InboundUser] {
	service := vless.NewService[*adapter.InboundUser](h.logger, adapter.NewUpstreamContextHandler(h.newConnectionEx, h.newPacketConnectionEx))
	service.UpdateUsers(adapter.NewInboundUsers(users, func(it option.VLESSUser) string {
		return it.Name
	}), common.Map(users, func(it option.VLESSUser) string {
		return it.UUID
	}), common.Map(users, func(it option.VLESSUser) string {
		return it.Flow
	}))
	return service
}

func (h *Inbound) UpdateUsers(users []option.VLESSUser) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	h.service.Store(h.newService(users))
	h.users = users
	return nil
}
