package fbx

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/mdns"
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

	QueryApiVersion int `json:"-"`
}

const (
	apiVersionURL = "http://mafreebox.freebox.fr/api_version"
	mdnsService   = "_fbx-api._tcp"
)

type FreeboxDiscovery int

const (
	// FreeboxDiscoveryHTTP Freebox discovery by call to http://mafreebox.freebox.fr/api_version
	FreeboxDiscoveryHTTP FreeboxDiscovery = iota
	// FreeboxDiscoveryMDNS Freebox discovery by mDNS on service _fbx-api._tcp
	FreeboxDiscoveryMDNS
)

func NewFreeboxAPIVersion(client *FreeboxHttpClient, discovery FreeboxDiscovery, forceApiVersion int) (*FreeboxAPIVersion, error) {
	result := &FreeboxAPIVersion{}

	if err := result.getDiscovery(discovery)(client); err != nil {
		return nil, err
	}

	if err := result.setQueryApiVersion(forceApiVersion); err != nil {
		return nil, err
	}

	if !result.IsValid() {
		return nil, errors.New("could not get the API version")
	}
	log.Debug.Println("APIVersion", result)
	return result, nil
}

func (f *FreeboxAPIVersion) IsValid() bool {
	if f == nil {
		return false
	}
	return f.APIDomain != "" &&
		f.UID != "" &&
		f.HTTPSAvailable &&
		f.HTTPSPort != 0 &&
		f.DeviceName != "" &&
		f.APIVersion != "" &&
		f.APIBaseURL != "" &&
		f.DeviceType != "" &&
		f.QueryApiVersion > 0
}

func (f *FreeboxAPIVersion) GetURL(path string, miscPath ...interface{}) (string, error) {
	if !f.IsValid() {
		return "", errors.New("invalid FreeboxAPIVersion")
	}
	args := make([]interface{}, len(miscPath)+4)
	args[0] = f.APIDomain
	args[1] = f.HTTPSPort
	args[2] = f.APIBaseURL
	args[3] = f.QueryApiVersion
	if len(miscPath) > 0 {
		copy(args[4:], miscPath)
	}
	return fmt.Sprintf("https://%s:%d%sv%d/"+path, args...), nil
}

func (f *FreeboxAPIVersion) getDiscovery(discovery FreeboxDiscovery) func(client *FreeboxHttpClient) error {
	function := func(*FreeboxHttpClient) error {
		return errors.New("wrong discovery argument")
	}

	switch discovery {
	case FreeboxDiscoveryHTTP:
		function = f.newFreeboxAPIVersionHTTP
	case FreeboxDiscoveryMDNS:
		function = f.newFreeboxAPIVersionMDNS
	default:
	}

	return function
}

func (f *FreeboxAPIVersion) newFreeboxAPIVersionHTTP(client *FreeboxHttpClient) error {
	log.Info.Println("Freebox discovery: GET", apiVersionURL)

	// HTTP GET api version
	r, err := client.client.Get(apiVersionURL)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(f); err != nil {
		return err
	}
	return nil
}

func (f *FreeboxAPIVersion) newFreeboxAPIVersionMDNS(*FreeboxHttpClient) error {
	log.Info.Println("Freebox discovery: mDNS")
	entries := make(chan *mdns.ServiceEntry, 4)

	// mDNS lookup
	go func() {
		defer close(entries)
		if err := mdns.Lookup(mdnsService, entries); err != nil {
			log.Error.Println("mDNS lookup failed:", err)
		}
		log.Debug.Println("End of mDNS lookup")
	}()

	for entry := range entries {
		deviceName := entry.Name
		idx := strings.Index(deviceName, ".")
		if idx >= 0 {
			deviceName = deviceName[0:idx]
		}
		deviceName = strings.ReplaceAll(deviceName, "\\", "")

		*f = FreeboxAPIVersion{
			DeviceName: deviceName,
		}
		for i := range entry.InfoFields {
			kv := strings.SplitN(entry.InfoFields[i], "=", 2)
			if len(kv) != 2 {
				break
			}
			switch kv[0] {
			case "api_domain":
				f.APIDomain = kv[1]
			case "uid":
				f.UID = kv[1]
			case "https_available":
				f.HTTPSAvailable = (kv[1] == "1")
			case "https_port":
				port, _ := strconv.ParseUint(kv[1], 10, 16)
				f.HTTPSPort = uint16(port)
			case "api_version":
				f.APIVersion = kv[1]
			case "api_base_url":
				f.APIBaseURL = kv[1]
			case "device_type":
				f.DeviceType = kv[1]
			default:
			}
		}
		if f.IsValid() {
			return nil
		}
	}

	return errors.New("MDNS timeout")
}

func (f *FreeboxAPIVersion) setQueryApiVersion(forceApiVersion int) error {
	versionSplit := strings.Split(f.APIVersion, ".")
	if len(versionSplit) != 2 {
		return fmt.Errorf("could not decode the api version \"%s\"", f.APIVersion)
	}
	if apiVersionFromDiscovery, err := strconv.Atoi(versionSplit[0]); err != nil {
		return err
	} else if forceApiVersion > apiVersionFromDiscovery {
		return fmt.Errorf("could use the api version %d which is higher than %d", forceApiVersion, apiVersionFromDiscovery)
	} else if forceApiVersion > 0 {
		f.QueryApiVersion = forceApiVersion
	} else {
		f.QueryApiVersion = apiVersionFromDiscovery
	}
	return nil
}
