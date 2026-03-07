package handler

const (
	// BaseLayout is the default path for layout templates.
	BaseLayout = "layouts/base"

	// RootPath is the root path the route group.
	RootPath = "/"

	// ErrNilACDFatalLogMsg is used if app or cfg or db var pointer is nil.
	ErrNilACDFatalLogMsg = "app, cfg or db is nil"

	// OrderNameASC is the GORM order clause for sorting by name ascending.
	OrderNameASC = "name ASC"

	// OrderUsernameASC is the GORM order clause for sorting by username ascending.
	OrderUsernameASC = "username ASC"
)
