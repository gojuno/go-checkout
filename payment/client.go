package payment

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gojuno/go-checkout"
)

// Caller makes HTTP call with given options and decode response into given struct.
// checkout.Client implements this interface. You may create payment.Client with own caller for test purposes.
type Caller interface {
	Call(ctx context.Context, method, path, idempotencyKey string, reqObj, respObj interface{}) (statusCode int, callErr error)
}

// Client is a client for work with Payment entity.
// https://docs.checkout.com/v2.0/docs/payments-quickstart
type Client struct {
	caller Caller
}

type SourceType string

type SourceScheme string

type CardType string

type Status string

type Source struct {
	ID             string         `json:"id"`
	Type           SourceType     `json:"type"`
	BillingAddress BillingAddress `json:"billing_address"`
	ExpiryMonth    uint           `json:"expiry_month"`
	ExpiryYear     uint           `json:"expiry_year"`
	Name           string         `json:"name"`
	Scheme         SourceScheme   `json:"scheme"`
	Last4          string         `json:"last4"`
	Fingerprint    string         `json:"fingerprint"`
	BIN            string         `json:"bin"`
	CardType       CardType       `json:"card_type"`
	CardCategory   string         `json:"card_category"`
	Issuer         string         `json:"issuer"`
	IssuerCountry  string         `json:"issuer_country"`
	ProductID      string         `json:"product_id"`
	ProductType    string         `json:"product_type"`
	AVSCheck       string         `json:"avs_check"`
	CVVCheck       string         `json:"cvv_check"`
}

type BillingAddress struct {
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	ZIP          string `json:"zip"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
}

type Customer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Risk struct {
	Flagged bool `json:"flagged"`
}

type Payment struct {
	ID              string    `json:"id"`
	ActionID        string    `json:"action_id"`
	Amount          uint      `json:"amount"`
	Currency        string    `json:"currency"`
	Approved        bool      `json:"approved"`
	Status          Status    `json:"status"`
	AuthCode        string    `json:"auth_code"`
	ECI             string    `json:"eci"`
	SchemeID        string    `json:"scheme_id"`
	ResponseCode    string    `json:"response_code"`
	ResponseSummary string    `json:"response_summary"`
	Risk            Risk      `json:"risk"`
	Source          Source    `json:"source"`
	Customer        Customer  `json:"customer"`
	ProcessedOn     time.Time `json:"processed_on"`
	Reference       string    `json:"reference"`
}

type CreationSource struct {
	Type        SourceType `json:"type"`                   // possible values: card, token, id
	ID          string     `json:"id,omitempty"`           // specify, if type is "id"
	Token       string     `json:"token,omitempty"`        // specify, if type is "token"
	Number      string     `json:"number,omitempty"`       // specify, if type is "card"
	ExpiryMonth uint       `json:"expiry_month,omitempty"` // specify, if type is "card"
	ExpiryYear  uint       `json:"expiry_year,omitempty"`  // specify, if type is "card"
	CVV         string     `json:"cvv,omitempty"`
}

type CreateParams struct {
	Source      CreationSource         `json:"source"`
	Amount      uint                   `json:"amount"`
	Currency    string                 `json:"currency"`
	Capture     *bool                  `json:"capture,omitempty"`
	Description string                 `json:"description,omitempty"`
	Reference   string                 `json:"reference,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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

type Error struct {
	Reason string
}

// Error implements error interface.
func (e Error) Error() string {
	return fmt.Sprintf("payment error reason: %s", e.Reason)
}

const (
	StatusAuthorized   Status = "Authorized"
	StatusCaptured     Status = "Captured"
	StatusCardVerified Status = "Card Verified"
	StatusDeclined     Status = "Declined"
	StatusPending      Status = "Pending"

	SourceTypeCard  SourceType = "card"
	SourceTypeToken SourceType = "token"
	SourceTypeID    SourceType = "id"

	SourceSchemeVisa            SourceScheme = "Visa"
	SourceSchemeMastercard      SourceScheme = "Mastercard"
	SourceSchemeAmericanExpress SourceScheme = "American Express"
	SourceSchemeJCB             SourceScheme = "JCB"
	SourceSchemeDinersClub      SourceScheme = "Diners Club International"
	SourceSchemeDiscover        SourceScheme = "Discover"

	CardTypeCredit  CardType = "Credit"
	CardTypeDebit   CardType = "Debit"
	CardTypePrepaid CardType = "Prepaid"
	CardTypeCharge  CardType = "Charge"

	paymentsPath = "/payments"
)

var (
	ErrPaymentNotFound   = Error{Reason: "Payment not found"}
	ErrVoidNotAllowed    = Error{Reason: "Void not allowed"}
	ErrRefundNotAllowed  = Error{Reason: "Refund not allowed"}
	ErrCaptureNotAllowed = Error{Reason: "Capture not allowed"}
)

func NewClient(caller Caller) *Client {
	return &Client{
		caller: caller,
	}
}

// Create creates new payment
// Using token: https://docs.checkout.com/v2.0/docs/request-a-card-payment
// Using existing card: https://docs.checkout.com/v2.0/docs/use-an-existing-card
func (c *Client) Create(ctx context.Context, idempotencyKey string, params *CreateParams) (*Payment, error) {
	payment := &Payment{}
	statusCode, err := c.caller.Call(ctx, "POST", paymentsPath, idempotencyKey, params, payment)
	if err != nil {
		return nil, err
	}

	switch statusCode {
	case http.StatusCreated, http.StatusAccepted:
		return payment, nil
	default:
		return nil, checkout.UnknownError{StatusCode: statusCode}
	}
}

// Void cancels a non-captured payment
// https://docs.checkout.com/v2.0/docs/void-a-payment
func (c *Client) Void(ctx context.Context, paymentID string, params *VoidParams) error {
	statusCode, err := c.caller.Call(ctx, "POST", fmt.Sprintf("%s/%s/voids", paymentsPath, paymentID), "", params, nil)
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
		return checkout.UnknownError{StatusCode: statusCode}
	}
}

// Refund refunds a captured payment
// https://docs.checkout.com/v2.0/docs/refund-a-payment
func (c *Client) Refund(ctx context.Context, paymentID string, idempotencyKey string, params *RefundParams) error {
	statusCode, err := c.caller.Call(ctx, "POST", fmt.Sprintf("%s/%s/refunds", paymentsPath, paymentID), idempotencyKey, params, nil)
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
		return checkout.UnknownError{StatusCode: statusCode}
	}

}

// Capture captures a non-captured payment
// https://docs.checkout.com/v2.0/docs/capture-a-payment
func (c *Client) Capture(ctx context.Context, paymentID string, params *CaptureParams) error {
	statusCode, err := c.caller.Call(ctx, "POST", fmt.Sprintf("%s/%s/captures", paymentsPath, paymentID), "", params, nil)
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
		return checkout.UnknownError{StatusCode: statusCode}
	}
}
