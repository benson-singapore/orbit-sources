//go:build !wasm

package host

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPGet performs an HTTP GET (native dev without WASM host).
func HTTPGet(url string, headers map[string]string) ([]byte, int, error) {
	return doHTTP(http.MethodGet, url, headers, "")
}

// HTTPPost performs an HTTP POST (native dev without WASM host).
func HTTPPost(url string, headers map[string]string, body string) ([]byte, int, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}
	return doHTTP(http.MethodPost, url, headers, body)
}

func doHTTP(method, url string, headers map[string]string, body string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, 0, err
	}
	return data, resp.StatusCode, nil
}

// NowUnix returns the current unix timestamp.
func NowUnix() int64 {
	return time.Now().Unix()
}
