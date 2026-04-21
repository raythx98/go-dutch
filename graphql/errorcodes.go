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
	UserAlreadyInGroup
	SameCurrencyConversion
	ConversionSourceLegDeletion
)

var Messages = map[int]string{
	EmailAlreadyRegistered:      "Email is already registered",
	UsernameAlreadyRegistered:   "Username is already registered",
	EmailNotRegistered:          "Email is not registered",
	InvalidCredentials:          "Invalid credentials",
	UserDoesNotExist:            "User does not exist",
	CurrencyNotSupported:        "Currency is not supported",
	NotGroupMember:              "You are not an existing group member",
	InvalidInviteCode:           "Invalid invite code",
	UserAlreadyInGroup:          "You are already in the group",
	SameCurrencyConversion:      "Source and target currencies must be different",
	ConversionSourceLegDeletion: "Cannot delete a conversion source leg directly — delete the conversion expense instead",
}
