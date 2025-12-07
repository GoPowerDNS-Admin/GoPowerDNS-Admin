package config

// DB holds the database configuration settings.
type DB struct {
	Extras     string
	Host       string
	Port       int
	User       string
	Password   string
	Name       string
	GormEngine string
}
