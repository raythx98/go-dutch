package graphql

const (
	EmailAlreadyRegistered = iota
	UsernameAlreadyRegistered
	EmailNotRegistered
	InvalidCredentials
	UserDoesNotExist
	CurrencyNotSupported
	NotGroupMember
	InvalidInviteCode
)

var Messages = map[int]string{
	EmailAlreadyRegistered:    "Email is already registered",
	UsernameAlreadyRegistered: "Username is already registered",
	EmailNotRegistered:        "Email is not registered",
	InvalidCredentials:        "Invalid credentials",
	UserDoesNotExist:          "User does not exist",
	CurrencyNotSupported:      "Currency is not supported",
	NotGroupMember:            "You are not an existing group member",
	InvalidInviteCode:         "Invalid invite code",
}
