package checkout

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// PaymentClient is a client for work with Payment entity.
// https://docs.checkout.com/v2.0/docs/payments-quickstart
type PaymentClient struct {
	Caller Caller
}

type PaymentStatus string

type PaymentSource struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	ExpiryMonth   uint   `json:"expiry_month"`
	ExpiryYear    uint   `json:"expiry_year"`
	Scheme        string `json:"scheme"`
	Last4         string `json:"last4"`
	Fingerprint   string `json:"fingerprint"`
	BIN           string `json:"bin"`
	CardType      string `json:"card_type"`
	CardCategory  string `json:"card_category"`
	Issuer        string `json:"issuer"`
	IssuerCountry string `json:"issuer_country"`
	ProductID     string `json:"product_id"`
	ProductType   string `json:"product_type"`
	AVSCheck      string `json:"avs_check"`
	CVVCheck      string `json:"cvv_check"`
}

type Customer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type PaymentRisk struct {
	Flagged bool `json:"flagged"`
}

type Payment struct {
	ID              string        `json:"id"`
	ActionID        string        `json:"action_id"`
	Amount          uint          `json:"amount"`
	Currency        uint          `json:"currency"`
	Approved        bool          `json:"approved"`
	Status          PaymentStatus `json:"status"`
	AuthCode        string        `json:"auth_code"`
	ECI             string        `json:"eci"`
	SchemeID        string        `json:"scheme_id"`
	ResponseCode    string        `json:"response_code"`
	ResponseSummary string        `json:"response_summary"`
	Risk            PaymentRisk   `json:"risk"`
	Source          PaymentSource `json:"source"`
	Customer        Customer      `json:"customer"`
	ProcessedOn     time.Time     `json:"processed_on"`
	Reference       string        `json:"reference"`
}

type CreateParams struct {
	Source struct {
		Type  string `json:"type"`
		ID    string `json:"id,omitempty"`
		Token string `json:"token,omitempty"`
	} `json:"source"`
	Amount    uint   `json:"amount"`
	Currency  uint   `json:"currency"`
	Reference string `json:"reference,omitempty"`
}

type VoidParams struct {
	Reference string                 `json:"reference,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type RefundParams struct {
	Amount    uint                   `json:"amount,omitempty"`
	Reference string                 `json:"reference,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type CaptureParams struct {
	Amount    uint                   `json:"amount,omitempty"`
	Reference string                 `json:"reference,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentError struct {
	Reason string
}

// Error implements error interface.
func (e PaymentError) Error() string {
	return fmt.Sprintf("payment error reason: %s", e.Reason)
}

const (
	PaymentStatusAuthorized   PaymentStatus = "Authorized"
	PaymentStatusCaptured     PaymentStatus = "Captured"
	PaymentStatusCardVerified PaymentStatus = "Card Verified"
	PaymentStatusDeclined     PaymentStatus = "Declined"
	PaymentStatusPending      PaymentStatus = "Pending"

	paymentsPath = "payments"
)

var (
	ErrPaymentNotFound   = PaymentError{Reason: "Payment not found"}
	ErrVoidNotAllowed    = PaymentError{Reason: "Void not allowed"}
	ErrRefundNotAllowed  = PaymentError{Reason: "Refund not allowed"}
	ErrCaptureNotAllowed = PaymentError{Reason: "Capture not allowed"}
)

// Create creates new payment
// Using token: https://docs.checkout.com/v2.0/docs/request-a-card-payment
// Using existing card: https://docs.checkout.com/v2.0/docs/use-an-existing-card
func (c *PaymentClient) Create(ctx context.Context, idempotencyKey string, params *CreateParams) (*Payment, error) {
	payment := &Payment{}
	statusCode, err := c.Caller.Call(ctx, "POST", paymentsPath, idempotencyKey, params, payment)
	if err != nil {
		return nil, err
	}

	switch statusCode {
	case http.StatusCreated, http.StatusAccepted:
		return payment, nil
	default:
		return nil, UnknownError{StatusCode: statusCode}
	}
}

// Void cancels a non-captured payment
// https://docs.checkout.com/v2.0/docs/void-a-payment
func (c *PaymentClient) Void(ctx context.Context, paymentID string, params *VoidParams) error {
	statusCode, err := c.Caller.Call(ctx, "POST", fmt.Sprintf("%s/%s/voids", paymentsPath, paymentID), "", params, nil)
	if err != nil {
		return err
	}

	switch statusCode {
	case http.StatusAccepted:
		return nil
	case http.StatusForbidden:
		return ErrVoidNotAllowed
	case http.StatusNotFound:
		return ErrPaymentNotFound
	default:
		return UnknownError{StatusCode: statusCode}
	}
}

// Refund refunds a captured payment
// https://docs.checkout.com/v2.0/docs/refund-a-payment
func (c *PaymentClient) Refund(ctx context.Context, paymentID string, params *RefundParams) error {
	statusCode, err := c.Caller.Call(ctx, "POST", fmt.Sprintf("%s/%s/refunds", paymentsPath, paymentID), "", params, nil)
	if err != nil {
		return err
	}

	switch statusCode {
	case http.StatusAccepted:
		return nil
	case http.StatusForbidden:
		return ErrRefundNotAllowed
	case http.StatusNotFound:
		return ErrPaymentNotFound
	default:
		return UnknownError{StatusCode: statusCode}
	}

}

// Capture captures a non-captured payment
// https://docs.checkout.com/v2.0/docs/capture-a-payment
func (c *PaymentClient) Capture(ctx context.Context, paymentID string, params *CaptureParams) error {
	statusCode, err := c.Caller.Call(ctx, "POST", fmt.Sprintf("%s/%s/captures", paymentsPath, paymentID), "", params, nil)
	if err != nil {
		return err
	}

	switch statusCode {
	case http.StatusAccepted:
		return nil
	case http.StatusForbidden:
		return ErrCaptureNotAllowed
	case http.StatusNotFound:
		return ErrPaymentNotFound
	default:
		return UnknownError{StatusCode: statusCode}
	}
}
