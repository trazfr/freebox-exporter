package fbx

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/trazfr/freebox-exporter/log"
)

type config struct {
	APIVersion *FreeboxAPIVersion `json:"api"`
	AppToken   string             `json:"app_token"`
}

type FreeboxConnection struct {
	client  *FreeboxHttpClient
	session *FreeboxSession
	config  config
}

func getField(i interface{}, fieldName string) interface{} {
	value := reflect.ValueOf(i)
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}
	field := value.FieldByName(fieldName)
	if field.IsValid() == false {
		return ""
	}
	return field.Interface()
}

/*
 * FreeboxConnection
 */

func NewFreeboxConnection() (*FreeboxConnection, error) {
	client := NewFreeboxHttpClient()
	apiVersion, err := NewFreeboxAPIVersionHTTP(client)
	if err != nil {
		return nil, err
	}
	appToken, err := GetAppToken(client, apiVersion)
	if err != nil {
		return nil, err
	}
	session, err := NewFreeboxSession(appToken, client, apiVersion)
	if err != nil {
		return nil, err
	}

	return &FreeboxConnection{
		client:  client,
		session: session,
		config: config{
			APIVersion: apiVersion,
			AppToken:   appToken,
		},
	}, nil
}

func NewFreeboxConnectionFromConfig(reader io.Reader) (*FreeboxConnection, error) {
	client := NewFreeboxHttpClient()
	config := config{}
	if err := json.NewDecoder(reader).Decode(&config); err != nil {
		return nil, err
	}
	if config.APIVersion.IsValid() == false {
		return nil, fmt.Errorf("Invalid api_version: %v", config.APIVersion)
	}
	if config.AppToken == "" {
		return nil, fmt.Errorf("Invalid app_token: %s", config.AppToken)
	}

	session, err := NewFreeboxSession(config.AppToken, client, config.APIVersion)
	if err != nil {
		return nil, err
	}

	return &FreeboxConnection{
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
	url, err := f.config.APIVersion.GetURL(path)
	if err != nil {
		return err
	}
	log.Debug.Println("GET", url, "retry:", retry)

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
	url, err := f.config.APIVersion.GetURL("login/logout/")
	if err != nil {
		return err
	}
	return f.client.Post(url, nil, nil, f.session.AddHeader)
}
