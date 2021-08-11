# go-authproxy

Reverse proxy for client authentication.

## How to use

```go
package main

import (
	"net/http"

	"github.com/c032/go-authproxy"
)

func main() {
	rp := &authproxy.ReverseHTTP{
		AuthenticateFunc: func(req *http.Request) (authproxy.ClientInfo, error) {
			isAllowed := false

			// TODO: Use `req` to determine whether the request should be
			// allowed or not.

			if !isAllowed {
				return nil, authproxy.ErrUnauthorized
			}

			// Optionally include information about the user.
			//
			// The proxy will add these headers to the request that's
			// sent to the destination server:
			//
			//     Internal-Id: 1
			//     Internal-Username: test
			ci := authproxy.ClientInfo{
				"id": "1",
				"username": "test",
			}

			return ci, nil
		},
		ForwardTo: authproxy.ReverseHTTPForwardDestination{
			URLPrefix: "http://internal.service.example/",
		},
	}

	err := http.ListenAndServe(":3000", rp)
	if err != nil {
		panic(err)
	}
}
```

## License

Apache 2.0
