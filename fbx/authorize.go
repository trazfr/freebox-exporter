package fbx

import "os"

// freeboxAuthorize is a fixed structure for authorization request
// https://dev.freebox.fr/sdk/os/login/#request-authorization
type freeboxAuthorize struct {
	AppID      string `json:"app_id"`
	AppName    string `json:"app_name"`
	AppVersion string `json:"app_version"`
	DeviceName string `json:"device_name"`
}

var (
	authorize = func() freeboxAuthorize {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		return freeboxAuthorize{
			AppID:      "com.github.trazfr.fboxexp",
			AppName:    "prometheus-freebox-exporter",
			AppVersion: "0.0.1",
			DeviceName: hostname,
		}
	}()
)

// getFreeboxAuthorize retrieves the static request to POST to POST /api/vX/login/authorize/
func getFreeboxAuthorize() *freeboxAuthorize {
	return &authorize
}
