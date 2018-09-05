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
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/trazfr/freebox-exporter/log"
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

type freeboxResponseBase struct {
	Success   bool   `json:"success"`
	Message   string `json:"msg"`
	ErrorCode string `json:"error_code"`
}

type freeboxResponseResultBase struct {
	Challenge    string `json:"challenge"`
	PasswordSalt string `json:"password_salt"`
}

const (
	apiVersionURL = "http://mafreebox.freebox.fr/api_version"
)

func getChallengeAuthorizationRequest() freeboxAuthorizePostRequest {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return freeboxAuthorizePostRequest{
		AppID:      "com.github.trazfr.fboxexp",
		AppName:    "prometheus-freebox-exporter",
		AppVersion: "0.0.1",
		DeviceName: hostname,
	}
}

func newHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:    NewTlsConfig(),
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
		Timeout: 10 * time.Second,
	}
}

func getField(i interface{}, fieldNames ...string) string {
	log.Debug.Println(i, fieldNames)
	if s, ok := i.(string); ok && len(fieldNames) == 0 {
		return s
	}

	fieldName := fieldNames[0]
	nextFields := fieldNames[1:]

	value := reflect.ValueOf(i)
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}
	field := value.FieldByName(fieldName)
	if field.IsValid() == false {
		return ""
	}
	return getField(field.Interface(), nextFields...)
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
		client:       newHttpClient(),
		API:          FreeboxAPIVersion{},
		AppToken:     "",
		challenge:    "",
		passwordSalt: "",
	}
}

func (f *FreeboxConnection) call(auth bool, method string, path string, pathFmt []interface{}, body interface{}, out interface{}) error {
	auth = auth && f.sessionToken != ""
	url, err := f.API.getURL(path, pathFmt)
	if err != nil {
		return err
	}
	log.Debug.Println(method, url)

	var bodyReader io.Reader
	{
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
	if auth {
		req.Header.Set("X-Fbx-App-Auth", f.sessionToken)
	}
	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return err
	}
	// if we have a result.challenge / result.password_salt, take it
	needSessionTokenRefresh := false
	if s := getField(out, "Result", "Challenge"); s != "" && s != f.challenge {
		f.challenge = s
		needSessionTokenRefresh = true
	}
	if s := getField(out, "Result", "PasswordSalt"); s != "" && s != f.passwordSalt {
		f.passwordSalt = s
		needSessionTokenRefresh = true
	}
	if needSessionTokenRefresh {
		if f.refreshSessionToken(false) == false {
			return errors.New("Could not refresh the session token")
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
		freeboxResponseBase
		Result struct {
			freeboxResponseResultBase
			LoggedIn bool `json:"logged_in"`
		} `json:"result"`
	}{}
	if err := f.getInternal(auth, &r, "login"); err != nil {
		return err
	}
	if auth && r.Result.LoggedIn != true {
		return errors.New("Not logged in")
	}
	return nil
}

func (f *FreeboxConnection) askAuthorization() error {
	postResponse := struct {
		freeboxResponseBase
		Result struct {
			freeboxResponseResultBase
			AppToken string `json:"app_token"`
			TrackID  int64  `json:"track_id"`
		} `json:"result"`
	}{}
	if err := f.post(getChallengeAuthorizationRequest(), &postResponse, "login/authorize"); err != nil {
		return err
	}
	if postResponse.Success == false {
		return fmt.Errorf("The POST to login/authorize/ failed: code=\"%v\" msg=\"%v\"", postResponse.ErrorCode, postResponse.Message)
	}
	f.AppToken = postResponse.Result.AppToken

	pending := false
	accessGranted := false
	for accessGranted == false {
		r := struct {
			freeboxResponseBase
			Result struct {
				freeboxResponseResultBase
				Status string `json:"status"`
			} `json:"result"`
		}{}
		if err := f.get(&r, "login/authorize/%d", postResponse.Result.TrackID); err != nil {
			return err
		}
		if postResponse.Success == false {
			return errors.New("The GET to login/authorize failed")
		}
		switch r.Result.Status {
		case "pending":
			if pending == false {
				fmt.Println("Please accept the login on the Freebox Server")
				pending = true
			}
			time.Sleep(10 * time.Second)
		case "granted":
			accessGranted = true
		default:
			return fmt.Errorf("Access is %s", r.Result.Status)
		}
	}
	return nil
}

func (f *FreeboxConnection) refreshSessionToken(force bool) (ok bool) {
	log.Debug.Println("Refresh session token. Force:", force)
	oldChallenge := f.challenge
	if force || f.challenge == "" {
		if err := f.test(false); err != nil {
			log.Error.Println("Refresh token:", err.Error())
			return false
		}
	}
	if f.challenge == "" {
		panic("No challenge...")
	}

	if f.AppToken != "" && oldChallenge != f.challenge {
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
			freeboxResponseBase
			Result struct {
				freeboxResponseResultBase
				SessionToken string `json:"session_token"`
			} `json:"result"`
		}{}
		f.sessionToken = ""
		if err := f.post(&req, &res, "login/session"); err == nil {
			log.Debug.Println("login result:", res.Success, "token:", res.Result.SessionToken)
			if res.Success && res.Result.SessionToken != "" {
				f.sessionToken = res.Result.SessionToken
				return true
			}
		}
		return false
	}
	return f.AppToken != ""
}

func (f *FreeboxConnection) Login() error {
	if f.API.isValid() == false {
		if err := f.API.refresh(f.client); err != nil {
			return err
		}
	}

	if f.refreshSessionToken(true) == true {
		return nil
	}

	err := f.askAuthorization()
	if err == nil {
		if f.refreshSessionToken(false) == false {
			return errors.New("Could not refresh the session token")
		}
	}
	return err
}

func (f *FreeboxConnection) Logout() error {
	res := freeboxResponseBase{}
	return f.post(nil, &res, "logout")
}
