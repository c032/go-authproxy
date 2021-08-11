package authproxy

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/c032/go-logger"
)

var ErrUnauthorized = errors.New("unauthorized")

type ReverseHTTPAuthenticateFunc func(req *http.Request) (ClientInfo, error)

const ReverseHTTPDefaultHeaderPrefix = "Internal-"

type ReverseHTTPForwardDestination struct {
	URLPrefix string
}

// ReverseHTTP is a reverse proxy.
//
// Public members should only be modified during initialization, before calling
// any struct methods.
type ReverseHTTP struct {
	HeaderPrefix string

	ForwardTo ReverseHTTPForwardDestination

	// Must be set by whoever creates the struct.
	AuthenticateFunc ReverseHTTPAuthenticateFunc

	Logger logger.Logger

	mu sync.Mutex
	c  *http.Client
}

func (r *ReverseHTTP) logger() logger.Logger {
	log := r.Logger

	if log == nil {
		return logger.Discard
	}

	return log
}

func (r *ReverseHTTP) client() *http.Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.c == nil {
		r.c = &http.Client{}
	}

	return r.c
}

func (r *ReverseHTTP) headerPrefix() string {
	headerPrefix := r.HeaderPrefix
	if headerPrefix == "" {
		headerPrefix = ReverseHTTPDefaultHeaderPrefix
	}

	return headerPrefix
}

func (r *ReverseHTTP) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log := r.logger()

	headerPrefix := r.headerPrefix()

	if !strings.HasSuffix(headerPrefix, "-") {
		panic("`headerPrefix` must end with a `-` character")
	}

	var (
		err error
		ci  ClientInfo
	)

	ci, err = r.AuthenticateFunc(req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var forwardURL *url.URL
	forwardURL, err = url.Parse(r.ForwardTo.URLPrefix)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
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
		if err == ErrUnauthorized {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			log.Print(err.Error())

			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	for k, values := range req.Header {
		if strings.HasPrefix(k, headerPrefix) {
			continue
		}

		for _, v := range values {
			forwardRequest.Header.Add(k, v)
		}
	}

	if ci != nil {
		for rawKey, v := range ci {
			k := headerPrefix + rawKey
			forwardRequest.Header.Set(k, v)
		}
	}

	httpClient := r.client()

	var resp *http.Response

	resp, err = httpClient.Do(forwardRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)

		return
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
		log.Print(err.Error())

		return
	}
}
