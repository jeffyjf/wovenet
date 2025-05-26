package database

type Config struct {
	// DatabaseType defines the type of database used (e.g., SQLite, MySQL, PostgreSQL).
	Driver string `mapstructure:"driver"`
	// DatabaseURL is the connection string for the database.
	Connection string `mapstructure:"connection"`
}
