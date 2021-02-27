package fbx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	client  http.Client
	ctx     context.Context
	decoder func(io.Reader, interface{}) error
}

type FreeboxHttpClientCallback func(*http.Request)

func jsonDecoder(reader io.Reader, out interface{}) error {
	return json.NewDecoder(reader).Decode(out)
}
func jsonDebugDecoder(reader io.Reader, out interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	log.Debug.Println("JSON data:", string(data))
	return jsonDecoder(bytes.NewBuffer(data), out)
}

func NewFreeboxHttpClient(debug bool) *FreeboxHttpClient {
	result := &FreeboxHttpClient{
		client: http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     newTLSConfig(),
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     10 * time.Minute,
			},
			Timeout: 10 * time.Second,
		},
		ctx:     context.Background(),
		decoder: jsonDecoder,
	}

	if debug {
		log.Debug.Println("Debug enabled")
		result.decoder = jsonDebugDecoder
	}

	return result
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
	defer res.Body.Close()
	if err != nil {
		return err
	}

	r := struct {
		Success   bool        `json:"success"`
		Message   string      `json:"msg"`
		ErrorCode string      `json:"error_code"`
		Result    interface{} `json:"result"`
	}{
		Result: out,
	}
	if err := f.decoder(res.Body, &r); err != nil {
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
