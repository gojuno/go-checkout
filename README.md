# Checkout.com API client [![GoDoc](https://godoc.org/github.com/gojuno/go-checkout?status.svg)](http://godoc.org/github.com/gojuno/go-checkout) [![Build Status](https://travis-ci.org/gojuno/go-checkout.svg?branch=master)](https://travis-ci.org/gojuno/go-checkout) [![Go Report Card](https://goreportcard.com/badge/github.com/gojuno/go-checkout)](https://goreportcard.com/report/github.com/gojuno/go-checkout)

This repo contains Checkout.com API client written in Go.

Checkout.com API documentation: https://docs.checkout.com/v2.0/docs/integration-options

Before using this client you need to register an account.

## How to install

Download package:
```
go get github.com/gojuno/go-checkout
```

Client uses `github.com/pkg/errors`, so you may need to download this package as well:
```
go get github.com/pkg/errors
```

Go modules are supported as well.

## How to use

To init client you will need `secret_key` which you can get from your Checkout.com account profile.
```
import "github.com/gojuno/go-checkout"
...
// Init client
client := checkout.New(
	checkout.OptSecretKey("your_secret_key"),
)

// Create new payment
payment, err := client.Payment().Create(
	context.Background(),
	"payment_idempotency_key",
	&checkout.CreateParams{
        Source:   checkout.Source{
            Type: checkout.SourceTypeID,
            ID:   "src_vjkl7cyod4zejpkk5dwpvla7ca",
        },
        Amount:   2000,
        Currency: "USD",
	},
)
```

## Custom HTTP client

By default client uses `http.DefaultClient`. You can set custom HTTP client using `checkout.OptHTTPClient` option:
```
httpClient := &http.Client{
	Timeout: time.Minute,
}

client := checkout.New(
	checkout.OptSecretKey("your_secret_key"),
	checkout.OptHTTPClient(httpClient),
)
```
You can use any HTTP client, implementing `checkout.HTTPClient` interface with method `Do(r *http.Request) (*http.Response, error)`. Built-in `net/http` client implements it, of course.

## Sandbox

Checkout.com supports sandbox and live environment. By default, client will use live endpoint. To use sandbox define sandbox endpoint when client is created.

```
client := checkout.New(
	checkout.OptSecretKey("your_secret_key"),
	checkout.OptEndpoint(checkout.EndpointSandbox),
)
```
