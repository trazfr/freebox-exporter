package fbx

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

const (
	apiVersionURL = "http://mafreebox.freebox.fr/api_version"
)

func NewFreeboxAPIVersionHTTP(client *FreeboxHttpClient) (*FreeboxAPIVersion, error) {
	log.Debug.Println("GET", apiVersionURL)

	// get api version
	r, err := client.client.Get(apiVersionURL)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	result := &FreeboxAPIVersion{}
	if err := json.NewDecoder(r.Body).Decode(result); err != nil {
		return nil, err
	}
	if result.IsValid() == false {
		return nil, errors.New("Could not get the API version")
	}
	log.Info.Println("APIVersion", result)
	return result, nil
}

func (f *FreeboxAPIVersion) IsValid() bool {
	if f == nil {
		return false
	}
	return f.APIDomain != "" &&
		f.UID != "" &&
		f.HTTPSAvailable == true &&
		f.HTTPSPort != 0 &&
		f.DeviceName != "" &&
		f.APIVersion != "" &&
		f.APIBaseURL != "" &&
		f.DeviceType != ""
}

func (f *FreeboxAPIVersion) GetURL(path string, miscPath ...interface{}) (string, error) {
	if f.IsValid() == false {
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
