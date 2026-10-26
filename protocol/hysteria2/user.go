package hysteria2

import (
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
)

func (h *Inbound) updateUsers(users []option.Hysteria2User) {
	h.service.UpdateUsers(adapter.NewInboundUsers(users, func(it option.Hysteria2User) string {
		return it.Name
	}), common.Map(users, func(it option.Hysteria2User) string {
		return it.Password
	}))
}

func (h *Inbound) UpdateUsers(users []option.Hysteria2User) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	h.updateUsers(users)
	return nil
}
