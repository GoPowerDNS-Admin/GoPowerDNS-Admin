package group

type formInput struct {
	Name        string   `validate:"required,min=1,max=255"`
	ExternalID  string   `validate:"max=512,required_unless=Source local"`
	Source      string   `validate:"required,oneof=local oidc ldap"`
	Description string   `validate:"max=255"`
	RoleID      uint
	UserIDs     []string // form values are strings
}
