package database

import (
	"sync"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var once sync.Once
var dbInstance *gorm.DB

func init() {
	// Ensure that viper is configured before calling GetDatabase
	// This could be done in the main application initialization
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.connection", "/var/lib/wovenet/wovenet.db")
	once.Do(func() {
		config := Config{}
		// Unmarshal the database configuration from viper
		err := viper.UnmarshalKey("database", &config)
		if err != nil {
			panic("failed to unmarshal database config: " + err.Error())
		}

		// Initialize the database connection based on the driver specified in the config
		switch config.Driver {
		case "sqlite":
			db, err := gorm.Open(sqlite.Open(config.Connection), &gorm.Config{})
			if err != nil {
				panic("failed to connect to the database: " + err.Error())
			}
			dbInstance = db
		default:
			panic("unsupported database driver: " + config.Driver)
		}
	})
}

func GetDatabase() *gorm.DB {
	// Ensure the database instance is initialized
	if dbInstance == nil {
		panic("database instance is not initialized")
	}
	return dbInstance
}
