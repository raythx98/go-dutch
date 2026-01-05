package graphql

const (
	EmailAlreadyRegistered = iota
	EmailNotRegistered
	InvalidCredentials
	UserDoesNotExist
	CurrencyNotSupported
	NotGroupMember
)

var Messages = map[int]string{
	EmailAlreadyRegistered: "Email is already registered",
	EmailNotRegistered:     "Email is not registered",
	InvalidCredentials:     "Invalid credentials",
	UserDoesNotExist:       "User does not exist",
	CurrencyNotSupported:   "Currency is not supported",
	NotGroupMember:         "You are not an existing group member",
}
