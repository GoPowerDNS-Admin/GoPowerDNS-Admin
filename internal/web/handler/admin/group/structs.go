package group

type formInput struct {
	Name        string   `validate:"required,min=1,max=100"`
	ExternalID  string   `validate:"max=255"`
	Source      string   `validate:"required,oneof=local oidc ldap"`
	Description string   `validate:"max=255"`
	RoleID      uint     `validate:"required"`
	UserIDs     []string // form values are strings
}
