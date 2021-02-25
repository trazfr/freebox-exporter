package fbx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/trazfr/freebox-exporter/log"
)

var (
	errAuthRequired = errors.New("auth_required")
	errInvalidToken = errors.New("invalid_token")
)

type FreeboxHttpClient struct {
	client http.Client
	ctx    context.Context
}

type FreeboxHttpClientCallback func(*http.Request)

func NewFreeboxHttpClient() *FreeboxHttpClient {
	return &FreeboxHttpClient{
		client: http.Client{
			Transport: &http.Transport{
				TLSClientConfig:    newTLSConfig(),
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
			Timeout: 10 * time.Second,
		},
		ctx: context.Background(),
	}
}

func (f *FreeboxHttpClient) Get(url string, out interface{}, callbacks ...FreeboxHttpClientCallback) error {
	req, err := http.NewRequestWithContext(f.ctx, "GET", url, nil)

	if err != nil {
		return err
	}
	for _, cb := range callbacks {
		cb(req)
	}
	return f.do(req, out)
}

func (f *FreeboxHttpClient) Post(url string, in interface{}, out interface{}, callbacks ...FreeboxHttpClientCallback) error {
	buffer := new(bytes.Buffer)
	if err := json.NewEncoder(buffer).Encode(in); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(f.ctx, "POST", url, buffer)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for _, cb := range callbacks {
		cb(req)
	}
	return f.do(req, out)
}

func (f *FreeboxHttpClient) do(req *http.Request, out interface{}) error {
	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	resBytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}
	log.Debug.Println("Response:", string(resBytes))

	r := struct {
		Success   bool        `json:"success"`
		Message   string      `json:"msg"`
		ErrorCode string      `json:"error_code"`
		Result    interface{} `json:"result"`
	}{
		Result: out,
	}
	if err := json.NewDecoder(bytes.NewBuffer(resBytes)).Decode(&r); err != nil {
		return err
	}
	if r.Success == false {
		switch r.ErrorCode {
		case errAuthRequired.Error():
			return errAuthRequired
		case errInvalidToken.Error():
			return errInvalidToken
		default:
			return fmt.Errorf("%s %s error_code=%s msg=%s", req.Method, req.URL, r.ErrorCode, r.Message)
		}
	}

	return nil
}
