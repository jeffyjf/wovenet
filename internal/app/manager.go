package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/kungze/wovenet/internal/logger"
	"github.com/kungze/wovenet/internal/tunnel"
)

type AppManager struct {
	ctx                   context.Context
	localExposedApps      map[string]*localApp
	remoteApps            sync.Map // map[localSocket]*remoteApp
	newConnectionCallback ClientConnectedCallback
}

func (am *AppManager) GetExposedApps() []LocalExposedApp {
	apps := []LocalExposedApp{}
	for _, app := range am.localExposedApps {
		apps = append(apps, LocalExposedApp{Name: app.config.AppName})
	}
	return apps
}

// TransferDataToLocalApp transfer data between tunnel stream and local app service
// appName the local app's name which the remote site's app client want to connect to
// socket if the local app's mode is range (usually have multiple socket), the socket specifies that connect to which socket
// stream tunnel stream
// remainingData the extra data except handshake data during handshake period, it come from remote site's app client, we need to
// write it to local app service
func (am *AppManager) TransferDataToLocalApp(appName string, socket string, stream tunnel.Stream, remainingData []byte) error {
	log := logger.GetDefault()
	app, ok := am.localExposedApps[appName]
	if !ok {
		log.Error("local app can not found", "localApp", appName)
		return fmt.Errorf("app: %s can not found", appName)
	}
	if err := app.StartDataConverter(stream, socket, remainingData); err != nil {
		log.Error("failed to start data converter", "error", err, "localApp", appName)
	}
	return nil
}

// ProcessNewRemoteSite when a new remote site connected successfully, we
// need to start the listeners for local sockets which for remote apps that
// located in this new remote site
func (am *AppManager) ProcessNewRemoteSite(ctx context.Context, remoteSite string, exposedApps []LocalExposedApp) {
	log := logger.GetDefault()
	am.remoteApps.Range(func(key, value interface{}) bool {
		remoteApp := value.(*remoteApp)
		if remoteApp.Active() {
			log.Info("remote app is already active, skip", "remoteSite", remoteSite, "remoteApp", remoteApp.config.AppName, "localSocket", remoteApp.config.LocalSocket)
			return true // continue to the next item
		}
		for _, exposedApp := range exposedApps {
			if remoteApp.config.SiteName == remoteSite && remoteApp.config.AppName == exposedApp.Name {
				if err := remoteApp.listen(ctx, am.newConnectionCallback); err != nil {
					log.Error("failed to start local socket listener for remote app", "localSocket", remoteApp.config.LocalSocket, "remoteSite", remoteSite, "appName", remoteApp.config.AppName, "error", err)
					continue
				}
			}
		}
		return true // continue to the next item
	})
}

// ProcessRemoteSiteGone when a remote site is disconnected, we need to
// stop the listeners which related to the remote apps
func (am *AppManager) ProcessRemoteSiteGone(remoteSite string) {
	log := logger.GetDefault()
	am.remoteApps.Range(func(key, value interface{}) bool {
		remoteApp := value.(*remoteApp)
		if remoteApp.config.SiteName == remoteSite {
			log.Info("stop local socket listener for remote app", "remoteSite", remoteSite, "remoteApp", remoteApp.config.AppName, "localSocket", remoteApp.config.LocalSocket)
			remoteApp.stop()
		}
		return true // continue to the next item
	})
}

func NewAppManager(ctx context.Context, localExposedApps []*LocalExposedAppConfig, remoteApps []*RemoteAppConfig, callback ClientConnectedCallback) (*AppManager, error) {
	am := AppManager{ctx: ctx, localExposedApps: map[string]*localApp{}, newConnectionCallback: callback}
	for _, exposedApp := range localExposedApps {
		a := newLocalApp(*exposedApp)
		am.localExposedApps[exposedApp.AppName] = a
	}

	// Delete remote apps from database that are not in the config file
	configApps, err := GetRemoteApps(map[string]any{"from_config": true})
	if err != nil {
		return nil, fmt.Errorf("failed to get remote apps from database: %w", err)
	}
	for _, configApp := range configApps {
		needDelete := true
		for _, remoteApp := range remoteApps {
			if configApp.SiteName == remoteApp.SiteName && configApp.AppName == remoteApp.AppName && configApp.LocalSocket == remoteApp.LocalSocket && configApp.AppSocket == remoteApp.AppSocket {
				needDelete = false
				am.remoteApps.Store(configApp.ID, newRemoteApp(*remoteApp))
			}
		}
		if needDelete {
			if err := DelRemoteApp(configApp.ID); err != nil {
				return nil, fmt.Errorf("failed to delete remote app %s from database: %w", configApp.AppName, err)
			}
		}
	}
	// Add remote apps from config file to the database if they are not already present
	for _, remoteApp := range remoteApps {
		configApp, err := GetRemoteApps(map[string]any{"from_config": true, "local_socket": remoteApp.LocalSocket})
		if err != nil {
			return nil, fmt.Errorf("failed to get remote app %s from database: %w", remoteApp.AppName, err)
		}
		if len(configApp) == 0 {
			// If the remote app is not in the database, we create it
			newApp, err := AddRemoteApp(*remoteApp)
			if err != nil {
				return nil, fmt.Errorf("failed to add remote app %s to database: %w", remoteApp.AppName, err)
			}
			am.remoteApps.Store(newApp.ID, newRemoteApp(newApp.RemoteAppConfig))
		}

	}

	return &am, nil
}
