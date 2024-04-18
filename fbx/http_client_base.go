package fbx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/trazfr/freebox-exporter/log"
)

var (
	errAuthRequired = errors.New("auth_required")
	errInvalidToken = errors.New("invalid_token")
)

type FreeboxHttpClientBase struct {
	client HttpClientInternal
	ctx    context.Context
}

type freeboxAPIResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"msg"`
	ErrorCode string `json:"error_code"`
}

func NewFreeboxHttpClientBase(client HttpClientInternal) FreeboxHttpClient {
	result := &FreeboxHttpClientBase{
		client: client,
		ctx:    context.Background(),
	}

	return result
}

func (f *FreeboxHttpClientBase) Get(url string, out interface{}, callbacks ...FreeboxHttpClientCallback) error {
	req, err := http.NewRequestWithContext(f.ctx, "GET", url, nil)

	if err != nil {
		return err
	}
	for _, cb := range callbacks {
		cb(req)
	}
	return f.do(req, out)
}

func (f *FreeboxHttpClientBase) Post(url string, in interface{}, out interface{}, callbacks ...FreeboxHttpClientCallback) error {
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

func (f *FreeboxHttpClientBase) do(req *http.Request, out interface{}) error {
	log.Debug.Println("HTTP request:", req.Method, req.URL.Path)

	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	var body []byte
	{
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
	}
	log.Debug.Println("HTTP Result:", string(body))

	apiResponse := freeboxAPIResponse{}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return err
	}
	if !apiResponse.Success {
		switch apiResponse.ErrorCode {
		case errAuthRequired.Error():
			return errAuthRequired
		case errInvalidToken.Error():
			return errInvalidToken
		default:
			return fmt.Errorf("%s %s error_code=%s msg=%s", req.Method, req.URL, apiResponse.ErrorCode, apiResponse.Message)
		}
	}

	result := struct {
		Result interface{} `json:"result"`
	}{
		Result: out,
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	return nil
}
