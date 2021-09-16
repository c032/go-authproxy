package authproxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type ForwarderRequestConfigureFunc func(req *http.Request) error

type Forwarder interface {
	Forward(w http.ResponseWriter, req *http.Request, configure ForwarderRequestConfigureFunc) error
}

type HTTPBaseURLForwarder struct {
	BaseURL string

	mu sync.Mutex
	c  *http.Client
}

func (f *HTTPBaseURLForwarder) client() *http.Client {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.c == nil {
		f.c = &http.Client{}
	}

	return f.c
}

func (f *HTTPBaseURLForwarder) Forward(w http.ResponseWriter, req *http.Request, configure ForwarderRequestConfigureFunc) error {
	var (
		err error

		forwardURL *url.URL
	)

	forwardURL, err = url.Parse(f.BaseURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return err
	}

	if !strings.HasSuffix(forwardURL.Path, "/") {
		forwardURL.Path += "/"
	}

	if req.URL.Path != "/" {
		forwardURL.Path += req.URL.Path[1:]
	}

	forwardURL.RawQuery = req.URL.RawQuery

	var forwardRequest *http.Request

	forwardRequest, err = http.NewRequest(req.Method, forwardURL.String(), req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return fmt.Errorf("could not create request: %w", err)
	}

	err = configure(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return fmt.Errorf("could not configure request: %w", err)
	}

	httpClient := f.client()

	var resp *http.Response

	resp, err = httpClient.Do(forwardRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)

		return fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	wHeaders := w.Header()
	for header, values := range resp.Header {
		wHeaders.Del(header)
		for _, value := range values {
			wHeaders.Add(header, value)
		}
	}

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("could not write: %w", err)
	}

	return nil
}
