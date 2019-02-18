package checkout

import (
	"encoding/json"
	"fmt"
)

// ServerError represents possible server error.
// Used status codes: https://docs.checkout.com/v2.0/docs/response-codes
type ServerError struct {
	StatusCode int
	Response   *ErrorResponse
}

// Error implements error interface.
func (e ServerError) Error() string {
	return fmt.Sprintf("status code: %d, response: %s", e.StatusCode, e.Response)
}

// ErrorResponse represents API error response.
// https://docs.checkout.com/v2.0/docs/validation-errors
// https://docs.checkout.com/v2.0/docs/response-codes
type ErrorResponse struct {
	RequestID  string   `json:"request_id"`
	ErrorType  string   `json:"error_type"`
	ErrorCodes []string `json:"error_codes"`
}

func (e ErrorResponse) String() string {
	str, _ := json.Marshal(e)
	return string(str)
}

// UnknownError represents possible unknown error.
type UnknownError struct {
	StatusCode int
}

// Error implements error interface.
func (e UnknownError) Error() string {
	return fmt.Sprintf("unknown status code: %d", e.StatusCode)
}
