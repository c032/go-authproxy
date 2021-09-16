package authproxy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	HeaderPrefix string

	Forwarder Forwarder

	// Must be set by whoever creates the struct.
	AuthenticateFunc ReverseHTTPAuthenticateFunc

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
		if err == ErrUnauthorized {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			log.Print(err.Error())

			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	err = r.Forwarder.Forward(w, req, func(forwardRequest *http.Request) error {
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
