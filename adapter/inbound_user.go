package adapter

import F "github.com/sagernet/sing/common/format"

// InboundUser is the user identity protocol services attach to an
// authenticated connection.
//
// Inbounds key their protocol services by *InboundUser instead of a slice
// index: a connection authenticated just before a runtime user update then
// still resolves to the user it authenticated as, never to whoever took over
// that index (or to an index that is out of range in the new list).
type InboundUser struct {
	Index int
	Name  string
}

// NewInboundUsers builds one InboundUser per option user.
func NewInboundUsers[T any](users []T, name func(T) string) []*InboundUser {
	result := make([]*InboundUser, len(users))
	for index, user := range users {
		result[index] = &InboundUser{Index: index, Name: name(user)}
	}
	return result
}

// UserName returns the configured name, or "" for a nil or unnamed user.
func (u *InboundUser) UserName() string {
	if u == nil {
		return ""
	}
	return u.Name
}

// DisplayName returns the name, falling back to the index for unnamed users.
func (u *InboundUser) DisplayName() string {
	if u == nil {
		return ""
	}
	if u.Name != "" {
		return u.Name
	}
	return F.ToString(u.Index)
}
