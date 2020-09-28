package checks

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultHTTPTimeout = time.Duration(3) * time.Second
)

// HTTPConfig is used for configuring an HTTP check. The only required field is `URL`.
//
// "Method" is optional and defaults to `GET` if undefined.
//
// "StatusCode" is optional and defaults to `200`.
//
// "Client" is optional; if undefined, a new client will be created using "Timeout".
//
// "Timeout" is optional and defaults to "3s".
type HTTPConfig struct {
	URL        *url.URL      // Required
	StatusCode int           // Optional (default 200)
	Client     *http.Client  // Optional
	Timeout    time.Duration // Optional (default 3s)
}

type HTTP struct {
	Config *HTTPConfig
}

func NewHTTP(cfg *HTTPConfig) (*HTTP, error) {
	if cfg == nil {
		return nil, fmt.Errorf("passed in config cannot be nil")
	}

	if err := cfg.prepare(); err != nil {
		return nil, fmt.Errorf("unable to prepare given config: %v", err)
	}

	return &HTTP{
		Config: cfg,
	}, nil
}

func (h *HTTP) Status() (interface{}, error) {
	start := time.Now()
	resp, err := h.do()
	if err != nil {
		return nil, err
	}
	io.Copy(ioutil.Discard, resp.Body)
	// TODO FEATURE: Check response content
	resp.Body.Close()

	// Check if StatusCode matches
	if resp.StatusCode != h.Config.StatusCode {
		return nil, fmt.Errorf("received status code '%v' does not match expected status code '%v'",
			resp.StatusCode, h.Config.StatusCode)
	}

	return time.Since(start), nil
}

func (h *HTTP) do() (*http.Response, error) {

	req, err := http.NewRequest("GET", h.Config.URL.String(), nil)
	req.Close = true
	if err != nil {
		return nil, fmt.Errorf("unable to create new HTTP request for check: %v", err)
	}

	resp, err := h.Config.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errors during check request: %v", err)
	}

	return resp, nil
}

func (h *HTTPConfig) prepare() error {
	if h.URL == nil {
		return errors.New("URL cannot be nil")
	}

	// Default StatusCode to 200
	if h.StatusCode == 0 {
		h.StatusCode = http.StatusOK
	}

	if h.Timeout == 0 {
		h.Timeout = defaultHTTPTimeout
	}

	if h.Client == nil {
		h.Client = &http.Client{Timeout: h.Timeout}
	} else {
		h.Client.Timeout = h.Timeout
	}

	return nil
}
