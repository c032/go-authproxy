# Simple example

## Server

* Listen on `:3000`.
* Forward to `http://localhost:8000/`.

```sh
go run main.go
```

## Client

* Only requests with the HTTP header `Authorization: test` are
  considered valid.
* Any other request is stopped at the proxy with an error.

```sh
curl -H 'Authorization: test' 'http://localhost:3000/'
```
