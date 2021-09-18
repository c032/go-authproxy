package authproxy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/c032/go-logger"
)

var ErrUnauthorized = errors.New("unauthorized")

type ReverseHTTPAuthenticateFunc func(req *http.Request) (ClientInfo, error)

const ReverseHTTPDefaultHeaderPrefix = "Internal-"

// ReverseHTTP is a reverse proxy.
//
// Public members should only be modified during initialization, before calling
// any struct methods.
type ReverseHTTP struct {
	sync.RWMutex

	// HeaderPrefix is the prefix that will be prepended to all new headers
	// added to the forwarded request.
	//
	// If it's empty on first use, it's set to
	// `ReverseHTTPDefaultHeaderPrefix`.
	//
	// It must end with a `-` character, otherwise a panic will happen on first
	// use.
	HeaderPrefix string

	// Forwarder is the interface that sends the request to its next
	// destination.
	Forwarder Forwarder

	// Must be set by whoever creates the struct.
	AuthenticateFunc ReverseHTTPAuthenticateFunc

	// Logger is the logger used by the methods of this struct.
	//
	// If `nil`, all logging is discarded.
	Logger logger.Logger
}

func (r *ReverseHTTP) logger() logger.Logger {
	log := r.Logger

	if log == nil {
		return logger.Discard
	}

	return log
}

func (r *ReverseHTTP) headerPrefix() string {
	log := r.logger()

	headerPrefix := r.HeaderPrefix
	if headerPrefix == "" {
		headerPrefix = ReverseHTTPDefaultHeaderPrefix

		log.WithFields(logger.Fields{
			"new_prefix": headerPrefix,
		}).Print("Updated `HeaderPrefix` because it was empty. Using default value.")
	}

	return headerPrefix
}

func (r *ReverseHTTP) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.RLock()
	forwarder := r.Forwarder
	r.RUnlock()

	if forwarder == nil {
		w.WriteHeader(http.StatusBadGateway)

		return
	}

	r.RLock()
	log := r.logger()
	r.RUnlock()

	r.Lock()
	headerPrefix := r.headerPrefix()
	r.Unlock()

	r.RLock()
	defer r.RUnlock()

	if !strings.HasSuffix(headerPrefix, "-") {
		panic("`headerPrefix` must end with a `-` character")
	}

	var (
		err error
		ci  ClientInfo
	)

	ci, err = r.AuthenticateFunc(req)
	if err != nil {
		if err == ErrUnauthorized {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			log.Print(err.Error())

			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	err = forwarder.Forward(w, req, func(forwardRequest *http.Request) error {
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

		return nil
	})
	if err != nil {
		err = fmt.Errorf("could not forward request: %w", err)

		log.Print(err.Error())
	}
}
