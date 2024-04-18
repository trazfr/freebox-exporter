package fbx

import (
	"encoding/json"
	"fmt"
	"io"
)

type config struct {
	APIVersion *FreeboxAPIVersion `json:"api"`
	AppToken   string             `json:"app_token"`
}

type FreeboxConnection struct {
	api     *FreeboxAPI
	client  *FreeboxHttpClient
	session *FreeboxSession
	config  config
}

/*
 * FreeboxConnection
 */

func NewFreeboxConnectionFromServiceDiscovery(discovery FreeboxDiscovery, forceApiVersion int) (*FreeboxConnection, error) {
	client := NewFreeboxHttpClient()
	api, err := NewFreeboxAPI(client, discovery, forceApiVersion)
	if err != nil {
		return nil, err
	}
	appToken, err := GetAppToken(client, api)
	if err != nil {
		return nil, err
	}
	session, err := NewFreeboxSession(appToken, client, api)
	if err != nil {
		return nil, err
	}

	return &FreeboxConnection{
		client:  client,
		session: session,
		config: config{
			APIVersion: api.apiVersion,
			AppToken:   appToken,
		},
	}, nil
}

func NewFreeboxConnectionFromConfig(reader io.Reader, forceApiVersion int) (*FreeboxConnection, error) {
	client := NewFreeboxHttpClient()
	config := config{}
	if err := json.NewDecoder(reader).Decode(&config); err != nil {
		return nil, err
	}
	queryVersion, err := config.APIVersion.getQueryApiVersion(forceApiVersion)
	if err != nil {
		return nil, err
	}
	if !config.APIVersion.IsValid() {
		return nil, fmt.Errorf("invalid api_version: %v", config.APIVersion)
	}
	if config.AppToken == "" {
		return nil, fmt.Errorf("invalid app_token: %s", config.AppToken)
	}

	api := &FreeboxAPI{
		apiVersion:   config.APIVersion,
		queryVersion: queryVersion,
	}
	session, err := NewFreeboxSession(config.AppToken, client, api)
	if err != nil {
		return nil, err
	}

	return &FreeboxConnection{
		api:     api,
		client:  client,
		session: session,
		config:  config,
	}, nil
}

func (f *FreeboxConnection) WriteConfig(writer io.Writer) error {
	return json.NewEncoder(writer).Encode(&f.config)
}

func (f *FreeboxConnection) get(path string, out interface{}) error {
	return f.getInternal(path, out, false)
}

func (f *FreeboxConnection) getInternal(path string, out interface{}, retry bool) error {
	url, err := f.api.GetURL(path)
	if err != nil {
		return err
	}

	if err := f.client.Get(url, out, f.session.AddHeader); err != nil {
		if retry {
			return err
		}

		switch err {
		case errAuthRequired, errInvalidToken:
			err := f.session.Refresh()
			if err != nil {
				return err
			}
			return f.getInternal(path, out, true)
		default:
			return err
		}
	}

	return nil
}

func (f *FreeboxConnection) Close() error {
	url, err := f.api.GetURL("login/logout/")
	if err != nil {
		return err
	}
	return f.client.Post(url, nil, nil, f.session.AddHeader)
}
