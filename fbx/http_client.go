package fbx

import (
	"net/http"
)

type FreeboxHttpClientCallback func(*http.Request)

type FreeboxHttpClient interface {
	Get(url string, out interface{}, callbacks ...FreeboxHttpClientCallback) error
	Post(url string, in interface{}, out interface{}, callbacks ...FreeboxHttpClientCallback) error
}
