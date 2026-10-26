package tuic

import (
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"

	"github.com/gofrs/uuid/v5"
)

func (h *Inbound) updateUsers(users []option.TUICUser) error {
	userUUIDList := make([][16]byte, 0, len(users))
	userPasswordList := make([]string, 0, len(users))
	for index, user := range users {
		if user.UUID == "" {
			return E.New("missing uuid for user ", index)
		}
		userUUID, err := uuid.FromString(user.UUID)
		if err != nil {
			return E.Cause(err, "invalid uuid for user ", index)
		}
		userUUIDList = append(userUUIDList, userUUID)
		userPasswordList = append(userPasswordList, user.Password)
	}
	h.server.UpdateUsers(adapter.NewInboundUsers(users, func(it option.TUICUser) string {
		return it.Name
	}), userUUIDList, userPasswordList)
	return nil
}

func (h *Inbound) UpdateUsers(users []option.TUICUser) error {
	h.userAccess.Lock()
	defer h.userAccess.Unlock()
	return h.updateUsers(users)
}
