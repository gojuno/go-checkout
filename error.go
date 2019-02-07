package checkout

import (
	"encoding/json"
	"fmt"
)

// CallError represents possible client error.
type CallError struct {
	StatusCode int
}

// Error implements error interface.
func (e *CallError) Error() string {
	return fmt.Sprintf("error status code: %d", e.StatusCode)
}

// ValidationError represents validation API error response.
// https://docs.checkout.com/v2.0/docs/validation-errors
type ValidationError struct {
	RequestID  string   `json:"request_id"`
	ErrorType  string   `json:"error_type"`
	ErrorCodes []string `json:"error_codes"`
}

// Error implements error interface.
func (e *ValidationError) Error() string {
	str, _ := json.Marshal(e)
	return string(str)
}
