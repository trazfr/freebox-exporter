package fbx

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/trazfr/freebox-exporter/log"
)

type config struct {
	APIVersion *FreeboxAPIVersion `json:"api"`
	AppToken   string             `json:"app_token"`
}

type FreeboxConnection struct {
	client FreeboxHttpClient
	config config
}

/*
 * FreeboxConnection
 */

func NewFreeboxConnectionFromServiceDiscovery(discovery FreeboxDiscovery, forceApiVersion int) (*FreeboxConnection, error) {
	clientInternal := httpClient()
	clientBase := NewFreeboxHttpClientBase(clientInternal)
	apiVersion, err := NewFreeboxAPIVersion(clientInternal, discovery)
	if err != nil {
		return nil, err
	}

	actualVersion, err := apiVersion.GetQueryApiVersion(forceApiVersion)
	if err != nil {
		return nil, err
	}
	appToken, err := getAppToken(clientBase, apiVersion, actualVersion)
	if err != nil {
		return nil, err
	}
	client, err := NewFreeboxSession(appToken, clientBase, apiVersion, actualVersion)
	if err != nil {
		return nil, err
	}

	return &FreeboxConnection{
		client: client,
		config: config{
			APIVersion: apiVersion,
			AppToken:   appToken,
		},
	}, nil
}

func NewFreeboxConnectionFromConfig(reader io.Reader, forceApiVersion int) (*FreeboxConnection, error) {
	client := NewFreeboxHttpClientBase(httpClient())
	config := config{}
	if err := json.NewDecoder(reader).Decode(&config); err != nil {
		return nil, err
	}
	queryVersion, err := config.APIVersion.GetQueryApiVersion(forceApiVersion)
	if err != nil {
		return nil, err
	}
	if !config.APIVersion.IsValid() {
		return nil, fmt.Errorf("invalid api_version: %v", config.APIVersion)
	}
	if config.AppToken == "" {
		return nil, fmt.Errorf("invalid app_token: %s", config.AppToken)
	}

	session, err := NewFreeboxSession(config.AppToken, client, config.APIVersion, queryVersion)
	if err != nil {
		return nil, err
	}

	return &FreeboxConnection{
		client: session,
		config: config,
	}, nil
}

// GetApiVersion get the connection info
func (f *FreeboxConnection) GetAPIVersion() *FreeboxAPIVersion {
	return f.config.APIVersion
}

func (f *FreeboxConnection) WriteConfig(writer io.Writer) error {
	return json.NewEncoder(writer).Encode(&f.config)
}

func (f *FreeboxConnection) Get(queryVersion int, path string, out interface{}) error {
	url, err := f.config.APIVersion.GetURL(queryVersion, path)
	if err != nil {
		return err
	}
	return f.client.Get(url, out)
}

func (f *FreeboxConnection) Logout(queryVersion int) error {
	url, err := f.config.APIVersion.GetURL(queryVersion, "login/logout/")
	if err != nil {
		return err
	}
	return f.client.Post(url, nil, nil)
}

func getAppToken(client FreeboxHttpClient, apiVersion *FreeboxAPIVersion, actualVersion int) (string, error) {
	reqStruct := getFreeboxAuthorize()
	postResponse := struct {
		AppToken string `json:"app_token"`
		TrackID  int64  `json:"track_id"`
	}{}

	url, err := apiVersion.GetURL(actualVersion, "login/authorize/")
	if err != nil {
		return "", err
	}

	if err := client.Post(url, reqStruct, &postResponse); err != nil {
		return "", err
	}

	counter := 0
	for {
		counter++
		status := struct {
			Status string `json:"status"`
		}{}

		url, err := apiVersion.GetURL(actualVersion, "login/authorize/%d", postResponse.TrackID)
		if err != nil {
			return "", err
		}
		client.Get(url, &status)

		switch status.Status {
		case "pending":
			log.Info.Println(counter, "Please accept the login on the Freebox Server")
			time.Sleep(10 * time.Second)
		case "granted":
			return postResponse.AppToken, nil
		default:
			return "", fmt.Errorf("access is %s", status.Status)
		}
	}
}
