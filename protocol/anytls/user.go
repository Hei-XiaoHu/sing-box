package anytls

import (
	anytls "github.com/anytls/sing-anytls"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
)

// newService builds an anytls service for users. Services are never mutated
// once in use: runtime updates build a new one and swap it in atomically.
func (h *Inbound) newService(users []option.AnyTLSUser) (*anytls.Service, error) {
	return anytls.NewService(anytls.ServiceConfig{
		Users: common.Map(users, func(it option.AnyTLSUser) anytls.User {
			return anytls.User(it)
		}),
		PaddingScheme: h.paddingScheme,
		Handler:       (*inboundHandler)(h),
		Logger:        h.logger,
	})
}

func (h *Inbound) UpdateUsers(users []option.AnyTLSUser) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	service, err := h.newService(users)
	if err != nil {
		return err
	}
	h.service.Store(service)
	return nil
}
