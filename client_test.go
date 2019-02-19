package checkout

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

type httpClientMock struct {
	do func(r *http.Request) (*http.Response, error)
}

type request struct {
	Field string `json:"field"`
}

type callerMock struct {
	t               *testing.T
	expectedMethod  string
	expectedPath    string
	expectedHeaders map[string]string
	expectedReqObj  interface{}
	returnRespObj   interface{}
	returnError     error
}

func (c *httpClientMock) Do(r *http.Request) (*http.Response, error) {
	return c.do(r)
}

func (c *callerMock) Call(ctx context.Context, method, path string, headers map[string]string, reqObj interface{}, respObj interface{}) error {
	if method != c.expectedMethod {
		c.t.Errorf("Invalid method: %s", method)
	}
	if path != c.expectedPath {
		c.t.Errorf("Invalid path: %s", path)
	}
	for k, v := range headers {
		if v != c.expectedHeaders[k] {
			c.t.Errorf("Invalid header %s: %s", k, v)
		}
	}
	if len(headers) != len(c.expectedHeaders) {
		c.t.Errorf("Invalid headers count: %d", len(headers))
	}

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		c.t.Fatalf("Marshal error: %s", err)
	}

	expectedReqBody, err := json.Marshal(c.expectedReqObj)
	if err != nil {
		c.t.Fatalf("Marshal error: %s", err)
	}

	if string(reqBody) != string(expectedReqBody) {
		c.t.Errorf("Invalid request body: %s", string(reqBody))
	}

	if c.returnRespObj != nil {
		reflect.ValueOf(respObj).Elem().Set(reflect.ValueOf(c.returnRespObj).Elem())
	}

	return c.returnError
}

func TestNew(t *testing.T) {
	c := New(
		OptSecretKey("secret_key"),
		OptHTTPClient(&httpClientMock{}),
	)

	if c == nil {
		t.Errorf("Client is nil")
	}
	if c.secretKey != "secret_key" {
		t.Errorf("Invalid secretKey: %s", c.secretKey)
	}
	if _, ok := c.httpClient.(*httpClientMock); !ok {
		t.Errorf("Invalid httpClient: %T", c.httpClient)
	}
}

func TestCall_WithResponse(t *testing.T) {
	httpClientMock := &httpClientMock{
		do: func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.checkout.com/somepath" {
				t.Errorf("Invalid request URI: %s", r.RequestURI)
			}
			if r.Method != "POST" {
				t.Errorf("Invalid request method: %s", r.Method)
			}
			if r.Header.Get(headerIdempotency) != "idempotency_key" {
				t.Errorf("Invalid request idempotency key: %s", r.Header.Get(headerIdempotency))
			}
			body, _ := ioutil.ReadAll(r.Body)
			if string(body) != `{"field":"request_value"}` {
				t.Errorf("Invalid request body: %s", string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"field":"response_value"}`)),
			}, nil
		},
	}

	req := request{
		Field: "request_value",
	}

	response := struct {
		Field string `json:"field"`
	}{}

	client := New(OptHTTPClient(httpClientMock), OptSecretKey("secret_key"))

	statusCode, err := client.Call(
		context.Background(),
		"POST",
		"/somepath",
		"idempotency_key",
		&req,
		&response,
	)

	if err != nil {
		t.Errorf("Call returned error: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Call returned unexpected status code: %d", statusCode)
	}

	if response.Field != "response_value" {
		t.Errorf("Response is invalid: %+v", response)
	}
}

func TestCall_WithServerError(t *testing.T) {
	httpClientMock := &httpClientMock{
		do: func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"category":"category_test"}`)),
			}, nil
		},
	}

	req := request{
		Field: "request_value",
	}

	client := New(OptHTTPClient(httpClientMock))

	statusCode, err := client.Call(
		context.Background(),
		"POST",
		"somepath",
		"",
		&req,
		nil,
	)

	if err == nil {
		t.Error("Call didn't return error")
	}
	if serverErr, ok := err.(ServerError); ok {
		if serverErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("Invalid error status code: %d", serverErr.StatusCode)
		}
	} else {
		t.Errorf("Call return invalid error type: %T", err)
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("Call returned unexpected status code: %d", statusCode)
	}
}

func TestCall_WithTransportError(t *testing.T) {
	httpClientMock := &httpClientMock{
		do: func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("do_error")
		},
	}

	req := request{
		Field: "request_value",
	}

	client := New(OptHTTPClient(httpClientMock))

	statusCode, err := client.Call(
		context.Background(),
		"POST",
		"somepath",
		"",
		&req,
		nil,
	)

	if err == nil {
		t.Error("Call didn't return error")
	}
	if errors.Cause(err).Error() != "do_error" {
		t.Errorf("Invalid error cause: %v", errors.Cause(err))
	}

	if statusCode != 0 {
		t.Errorf("Call returned unexpected status code: %d", statusCode)
	}
}
