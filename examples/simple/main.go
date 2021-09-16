package main

import (
	"net/http"
	"os"

	"github.com/c032/go-logger"

	"github.com/c032/go-authproxy"
)

func auth(req *http.Request) (authproxy.ClientInfo, error) {
	token := req.Header.Get("Authorization")
	if token != "test" {
		return nil, authproxy.ErrUnauthorized
	}

	ci := authproxy.ClientInfo{
		"user": "test",
		"id":   "1",
	}

	return ci, nil
}

func main() {
	const (
		listenAddr        = ":3000"
		forwardBaseURLStr = "http://localhost:8000/"
	)

	log := logger.Default

	rp := &authproxy.ReverseHTTP{
		Logger: log,

		AuthenticateFunc: auth,
		Forwarder: &authproxy.HTTPBaseURLForwarder{
			BaseURL: forwardBaseURLStr,
		},
	}

	err := http.ListenAndServe(listenAddr, rp)
	if err != nil {
		log.Print(err.Error())

		os.Exit(1)
	}
}
