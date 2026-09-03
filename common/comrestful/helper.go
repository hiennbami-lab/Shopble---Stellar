package comrestful

import (
	"context"
	"io"
	"net/http"
)

func methodSelector(method Method) string {
	switch method {
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case DELETE:
		return "DELETE"
	default:
		panic("invalid method")
	}
}

func newRequest(ctx context.Context, method, url string, data io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequestWithContext(ctx, method, url, data)
	if err != nil {
		return
	}
	return
}
