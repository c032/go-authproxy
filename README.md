# go-authproxy

Reverse proxy library for client authorization and authentication.

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
		Forwarder: authproxy.HTTPBaseURLForwarder{
			BaseURL:: "http://internal.service.example/",
		},
	}

	err := http.ListenAndServe(":3000", rp)
	if err != nil {
		panic(err)
	}
}
```

## Implementations of the `Forwarder` interface

### `HTTPBaseURLForwarder`

Given:

* Example request URL: `http://public.example/original/path`
* Example forwarder base URL: `http://internal.service.example/prefix`

The URLs are resolved with steps equivalent to these:

* Extract the path from the original request, without the leading slash:
  `original/path`.
* Ensure the forwarder base URL's path has a trailing slash: `/prefix`
  becomes `/prefix/`.
* Parse `http://internal.service.example/prefix/` into a variable, e.g.
  `forwardURL`.
* Create the final URL using `forwardURL.Parse("original/path")`.

## License

Apache 2.0
