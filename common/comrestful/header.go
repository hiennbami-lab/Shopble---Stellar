package comrestful

import "net/http"

type Header map[string]string

func (h Header) Attached(req *http.Request) {
	for key, element := range h {
		req.Header.Set(key, element)
	}
}
