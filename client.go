// Package checkout contains Go client for Checkout API.
//
// Checkout API documentation: https://docs.checkout.com/v2.0
package checkout

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// Caller makes HTTP call with given options and decode response into given struct.
// Client implements this interface and pass itself to entity clients. You may create entity clients with own caller for
// test purposes.
type Caller interface {
	Call(ctx context.Context, method, path string, idempotencyKey string, reqObj interface{}, respObj interface{}) (statusCode int, callErr error)
}

// HTTPClient is interface fot HTTP client. Built-in net/http.Client implements this interface as well.
type HTTPClient interface {
	Do(r *http.Request) (*http.Response, error)
}

// Option is a callback for redefine client parameters.
type Option func(*Client)

// Client contains API parameters and provides set of API entity clients.
type Client struct {
	httpClient HTTPClient
	endpoint   string
	secretKey  string
}

const (
	EndpointLive    = "https://api.checkout.com"
	EndpointSandbox = "https://api.sandbox.checkout.com"

	headerAuthorization = "Authorization"
	headerIdempotency   = "Cko-Idempotency-Key"
)

// New creates new client with given options.
func New(options ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		endpoint:   EndpointLive,
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// OptHTTPClient returns option with given HTTP client.
func OptHTTPClient(httpClient HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// OptSecretKey returns option with given secret key.
func OptSecretKey(secretKey string) Option {
	return func(c *Client) {
		c.secretKey = secretKey
	}
}

// OptEndpoint returns option with given API endpoint.
func OptEndpoint(endpoint string) Option {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// Call does HTTP request with given params using set HTTP client. Response will be decoded into respObj.
// ServerError may be returned if something went wrong. If API return error as response, then Call returns error of type checkout.ServerError.
func (c *Client) Call(ctx context.Context, method, path string, idempotencyKey string, reqObj interface{}, respObj interface{}) (statusCode int, callErr error) {
	var reqBody io.Reader

	if reqObj != nil {
		reqBodyBytes, err := json.Marshal(reqObj)
		if err != nil {
			return 0, errors.Wrap(err, "failed to marshal request body")
		}
		reqBody = bytes.NewBuffer(reqBodyBytes)
	}

	req, err := http.NewRequest(method, c.endpoint+path, reqBody)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create HTTP request")
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerAuthorization, c.secretKey)

	if idempotencyKey != "" {
		req.Header.Set(headerIdempotency, idempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "failed to do request")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			callErr = err
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= http.StatusBadRequest {
		switch {
		case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusTooManyRequests:
			return resp.StatusCode, ServerError{
				StatusCode: resp.StatusCode,
			}
		case resp.StatusCode >= http.StatusInternalServerError, resp.StatusCode == http.StatusTooManyRequests, resp.StatusCode == http.StatusUnprocessableEntity:
			var errorResponse ErrorResponse

			if err := json.Unmarshal(respBody, &errorResponse); err != nil {
				return resp.StatusCode, errors.Wrapf(err, "failed to unmarshal response error with status %d: %s", resp.StatusCode, string(respBody))
			}

			return resp.StatusCode, ServerError{
				StatusCode: resp.StatusCode,
				Response:   &errorResponse,
			}

		default:
			// All other 4xx codes should be handled by entity clients
			return resp.StatusCode, nil
		}
	}

	// Decode response into a struct if it was given
	if respObj != nil {
		if err := json.Unmarshal(respBody, respObj); err != nil {
			return resp.StatusCode, errors.Wrapf(err, "failed to unmarshal response body: %s", string(respBody))
		}
	}

	return resp.StatusCode, nil
}

// Payment creates client for work with corresponding entity.
func (c *Client) Payment() *PaymentClient {
	return &PaymentClient{caller: c}
}
