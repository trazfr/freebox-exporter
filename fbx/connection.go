package fbx

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/trazfr/freebox-exporter/log"
)

var (
	errAuthRequired = errors.New("auth_required")
	errInvalidToken = errors.New("invalid_token")
)

type FreeboxAPIVersion struct {
	APIDomain      string `json:"api_domain"`
	UID            string `json:"uid"`
	HTTPSAvailable bool   `json:"https_available"`
	HTTPSPort      uint16 `json:"https_port"`
	DeviceName     string `json:"device_name"`
	APIVersion     string `json:"api_version"`
	APIBaseURL     string `json:"api_base_url"`
	DeviceType     string `json:"device_type"`
}

type FreeboxConnection struct {
	client       *http.Client
	API          FreeboxAPIVersion `json:"api"`
	AppToken     string            `json:"app_token"`
	sessionToken string
	challenge    string
	passwordSalt string
}

type freeboxAuthorizePostRequest struct {
	AppID      string `json:"app_id"`
	AppName    string `json:"app_name"`
	AppVersion string `json:"app_version"`
	DeviceName string `json:"device_name"`
}

const (
	apiVersionURL = "http://mafreebox.freebox.fr/api_version"
)

var (
	authRequest *freeboxAuthorizePostRequest
)

func getChallengeAuthorizationRequest() *freeboxAuthorizePostRequest {
	if authRequest == nil {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		authRequest = &freeboxAuthorizePostRequest{
			AppID:      "com.github.trazfr.fboxexp",
			AppName:    "prometheus-freebox-exporter",
			AppVersion: "0.0.1",
			DeviceName: hostname,
		}
	}
	return authRequest
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:    newTlsConfig(),
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
		Timeout: 10 * time.Second,
	}
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

func (f *FreeboxAPIVersion) isValid() bool {
	return f.APIDomain != "" &&
		f.UID != "" &&
		f.HTTPSAvailable == true &&
		f.HTTPSPort != 0 &&
		f.DeviceName != "" &&
		f.APIVersion != "" &&
		f.APIBaseURL != "" &&
		f.DeviceType != ""
}

func (f *FreeboxAPIVersion) getURL(path string, miscPath []interface{}) (string, error) {
	if f.isValid() == false {
		return "", errors.New("Invalid FreeboxAPIVersion")
	}
	versionSplit := strings.Split(f.APIVersion, ".")
	if len(versionSplit) != 2 {
		return "", fmt.Errorf("Could not decode the api version \"%s\"", f.APIVersion)
	}
	args := make([]interface{}, len(miscPath)+4)
	args[0] = f.APIDomain
	args[1] = f.HTTPSPort
	args[2] = f.APIBaseURL
	args[3] = versionSplit[0]
	if len(miscPath) > 0 {
		copy(args[4:], miscPath)
	}
	return fmt.Sprintf("https://%s:%d%sv%s/"+path, args...), nil
}

func (f *FreeboxAPIVersion) refresh(client *http.Client) error {
	log.Debug.Printf("Get API version: %s\n", apiVersionURL)
	// get api version
	r, err := client.Get(apiVersionURL)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(f); err != nil {
		return err
	}
	if f.isValid() == false {
		return errors.New("Could not get the API version")
	}
	return nil
}

/*
 * FreeboxConnection
 */

func NewFreeboxConnection() *FreeboxConnection {
	return &FreeboxConnection{
		client: newHTTPClient(),
	}
}

func (f *FreeboxConnection) call(auth bool, method string, path string, pathFmt []interface{}, body interface{}, out interface{}) error {
	url, err := f.API.getURL(path, pathFmt)
	if err != nil {
		return err
	}
	log.Debug.Println(method, url)

	var bodyReader io.Reader
	if body != nil {
		buffer := new(bytes.Buffer)
		if err := json.NewEncoder(buffer).Encode(body); err != nil {
			return err
		}
		bodyReader = buffer
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && f.sessionToken != "" {
		req.Header.Set("X-Fbx-App-Auth", f.sessionToken)
	}
	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	resBytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}

	{
		buf := bytes.NewBuffer(resBytes)
		r := struct {
			Success   bool   `json:"success"`
			Message   string `json:"msg"`
			ErrorCode string `json:"error_code"`
			Result    struct {
				Challenge    string `json:"challenge"`
				PasswordSalt string `json:"password_salt"`
			} `json:"result"`
		}{}
		if err := json.NewDecoder(buf).Decode(&r); err != nil {
			return err
		}
		if r.Success == false {
			switch r.ErrorCode {
			case "auth_required":
				return errAuthRequired
			case "invalid_token":
				return errInvalidToken
			}
			return fmt.Errorf("%s %s error_code=%s msg=%s", method, url, r.ErrorCode, r.Message)
		}
		if f.challenge == "" || f.sessionToken != "" {
			f.refreshSessionToken(r.Result.Challenge)
		}
	}

	if out != nil {
		buf := bytes.NewBuffer(resBytes)
		r := struct {
			Result interface{} `json:"result"`
		}{
			Result: out,
		}
		if err := json.NewDecoder(buf).Decode(&r); err != nil {
			return err
		}
	}
	return nil

}

func (f *FreeboxConnection) get(out interface{}, path string, pathFmt ...interface{}) error {
	return f.getInternal(true, out, path, pathFmt...)
}

func (f *FreeboxConnection) getInternal(auth bool, out interface{}, path string, pathFmt ...interface{}) error {
	return f.call(auth, "GET", path, pathFmt, nil, out)
}

func (f *FreeboxConnection) post(in interface{}, out interface{}, path string, pathFmt ...interface{}) error {
	return f.call(true, "POST", path, pathFmt, in, out)
}

func (f *FreeboxConnection) test(auth bool) error {
	r := struct {
		LoggedIn bool `json:"logged_in"`
	}{}
	if err := f.getInternal(auth, &r, "login"); err != nil {
		return err
	}
	if auth && r.LoggedIn == false {
		return errors.New("Not logged in")
	}
	return nil
}

func (f *FreeboxConnection) askAuthorization() error {
	postResponse := struct {
		AppToken string `json:"app_token"`
		TrackID  int64  `json:"track_id"`
	}{}
	if err := f.post(getChallengeAuthorizationRequest(), &postResponse, "login/authorize"); err != nil {
		return err
	}
	f.AppToken = postResponse.AppToken

	counter := 0
	accessGranted := false
	for accessGranted == false {
		counter++
		r := struct {
			Status string `json:"status"`
		}{}
		if err := f.get(&r, "login/authorize/%d", postResponse.TrackID); err != nil {
			return err
		}
		switch r.Status {
		case "pending":
			fmt.Println(counter, "Please accept the login on the Freebox Server")
			time.Sleep(10 * time.Second)
		case "granted":
			accessGranted = true
		default:
			return fmt.Errorf("Access is %s", r.Status)
		}
	}
	return nil
}

func (f *FreeboxConnection) refreshSessionToken(challenge string) bool {
	if challenge != "" && challenge != f.challenge {
		f.challenge = challenge
		f.sessionToken = ""
	}
	if f.AppToken != "" && f.sessionToken == "" {
		hash := hmac.New(sha1.New, []byte(f.AppToken))
		hash.Write([]byte(f.challenge))
		password := hex.EncodeToString(hash.Sum(nil))

		authorizationRequest := getChallengeAuthorizationRequest()
		req := struct {
			AppID    string `json:"app_id"`
			Password string `json:"password"`
		}{
			AppID:    authorizationRequest.AppID,
			Password: password,
		}
		res := struct {
			SessionToken string `json:"session_token"`
		}{}
		f.sessionToken = ""
		if err := f.post(&req, &res, "login/session"); err != nil {
			return false
		}
		f.sessionToken = res.SessionToken
	}
	return f.sessionToken != ""
}

func (f *FreeboxConnection) Login() error {
	if f.API.isValid() == false {
		if err := f.API.refresh(f.client); err != nil {
			return err
		}
	}

	if f.AppToken == "" {
		err := f.askAuthorization()
		if err != nil {
			return err
		}
	} else {
		// refresh challenge
		if err := f.test(false); err != nil {
			return fmt.Errorf("Could not refresh the session token \"%v\"", err.Error())
		}
	}

	// test login
	if err := f.test(true); err != nil {
		return fmt.Errorf("Login test failed \"%v\"", err.Error())
	}

	if f.challenge == "" {
		f.AppToken = ""
		return f.Login()
	}

	return nil
}

func (f *FreeboxConnection) Logout() error {
	return f.post(nil, nil, "logout")
}
