package app

import (
	"fmt"

	"gorm.io/gorm"
)

type LocalExposedAppResponse struct {
	LocalExposedAppConfig
}

func (am *AppManager) GetLocalExposedApps() []LocalExposedAppResponse {
	apps := []LocalExposedAppResponse{}
	for _, app := range am.localExposedApps {
		apps = append(apps, LocalExposedAppResponse{LocalExposedAppConfig: app.config})
	}
	return apps
}

func (am *AppManager) ShowLocalExposedApp(appName string) *LocalExposedAppResponse {
	localApp, ok := am.localExposedApps[appName]
	if !ok {
		return nil
	}
	return &LocalExposedAppResponse{LocalExposedAppConfig: localApp.config}
}

type RemoteAppRequest struct {
	RemoteAppConfig
}

type RemoteAppResponse struct {
	RemoteAppModel
	Active bool `json:"active"`
}

func (am *AppManager) AddRemoteApp(remoteApp RemoteAppRequest, listen bool) (*RemoteAppResponse, error) {
	appObj, err := AddRemoteApp(remoteApp.RemoteAppConfig)
	if err != nil {
		return nil, err
	}
	appInstance := newRemoteApp(appObj.RemoteAppConfig)
	am.remoteApps.Store(appObj.ID, appInstance)
	if listen {
		if err := appInstance.listen(am.ctx, am.newConnectionCallback); err != nil {
			return nil, err
		}
	}
	return &RemoteAppResponse{RemoteAppModel: *appObj, Active: appInstance.Active()}, nil
}

func (am *AppManager) DelRemoteApp(appId uint) error {
	appObj, err := GetRemoteApp(appId)
	if err != nil {
		return err
	}
	if appObj.FromConfig {
		return fmt.Errorf("cannot delete remote app %s, it is from config file", appObj.AppName)
	}
	am.remoteApps.Range(func(key, value interface{}) bool {
		remoteApp := value.(*remoteApp)
		if remoteApp.config.LocalSocket == appObj.LocalSocket {
			remoteApp.stop()
			am.remoteApps.Delete(key)
			return false // stop iterating
		}
		return true // continue iterating
	})
	return DelRemoteApp(appId)
}

func (am *AppManager) GetRemoteApps() ([]RemoteAppResponse, error) {
	apps := []RemoteAppResponse{}
	var finalErr error
	am.remoteApps.Range(func(key, value interface{}) bool {
		remoteApp := value.(*remoteApp)
		appId := key.(uint)
		appObj, err := GetRemoteApp(appId)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				finalErr = err
				return false // stop iterating on error
			}
			// If we can't find the app in the database, stop and remote it from AppManager
			remoteApp.stop()
			am.remoteApps.Delete(key)
			return true // continue iterating
		}
		apps = append(apps, RemoteAppResponse{RemoteAppModel: *appObj, Active: remoteApp.Active()})
		return true
	})
	return apps, finalErr
}

func (am *AppManager) ShowRemoteApp(appId uint) *RemoteAppResponse {
	appObj, err := GetRemoteApp(appId)
	if err != nil {
		return nil
	}
	appInstance, ok := am.remoteApps.Load(appId)
	if !ok {
		return &RemoteAppResponse{RemoteAppModel: *appObj, Active: false}
	}
	return &RemoteAppResponse{RemoteAppModel: *appObj, Active: appInstance.(*remoteApp).Active()}
}
