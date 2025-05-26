package app

import (
	"fmt"
	"time"

	"github.com/kungze/wovenet/internal/database"
	"gorm.io/gorm"
)

type RemoteAppModel struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	CreatedAt       time.Time `json:"created_at"`
	FromConfig      bool      `json:"fromConfig" gorm:"default:false"` // Indicates if the app was created from a config file
	RemoteAppConfig `gorm:"embedded"`
}

func init() {
	db := database.GetDatabase()
	if err := db.AutoMigrate(
		&RemoteAppModel{},
	); err != nil {
		panic("failed to auto migrate app models: " + err.Error())
	}
}

// GetRemoteApps retrieves remote apps from the database based on the provided filter.
// The filter is a map where keys are column names and values are the values to filter by.
func GetRemoteApps(filter map[string]any) ([]RemoteAppModel, error) {
	var apps []RemoteAppModel
	db := database.GetDatabase()
	if err := db.Where(filter).Find(&apps).Error; err != nil {
		return nil, fmt.Errorf("failed to get remote apps: %w", err)
	}
	return apps, nil
}

// GetRemoteApp retrieves a single remote app by its ID.
func GetRemoteApp(appId uint) (*RemoteAppModel, error) {
	remoteApp := &RemoteAppModel{}
	if err := database.GetDatabase().First(remoteApp, appId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("remote app with ID %d not found", appId)
		}
		return nil, fmt.Errorf("failed to get remote app with ID %d: %w", appId, err)
	}
	return remoteApp, nil
}

func AddRemoteApp(app RemoteAppConfig) (*RemoteAppModel, error) {
	db := database.GetDatabase()
	if err := db.Create(&RemoteAppModel{
		RemoteAppConfig: app,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to add remote app %s: %w", app.AppName, err)
	}
	remoteApp := &RemoteAppModel{}
	if err := db.Where("local_socket = ?", app.LocalSocket).First(remoteApp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("remote app with local socket %s not found after creation", app.LocalSocket)
		}
		return nil, fmt.Errorf("failed to find remote app with local socket %s: %w", app.LocalSocket, err)
	}
	return remoteApp, nil
}

func DelRemoteApp(appId uint) error {
	db := database.GetDatabase()
	remoteApp := &RemoteAppModel{}
	if err := db.First(remoteApp, appId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // No error if the app is not found, just return nil
		}
		return fmt.Errorf("failed to find remote app with ID %d: %w", appId, err)
	}
	if remoteApp.FromConfig {
		return fmt.Errorf("cannot delete remote app %s, it was created from config file", remoteApp.AppName)
	}

	if err := db.Delete(&RemoteAppModel{}, appId).Error; err != nil {
		return fmt.Errorf("failed to delete remote app with ID %d: %w", appId, err)
	}
	return nil
}
