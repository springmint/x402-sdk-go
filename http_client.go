package x402

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

// HTTP header constants
const (
	PaymentSignatureHeader = "PAYMENT-SIGNATURE"
	PaymentRequiredHeader  = "PAYMENT-REQUIRED"
	PaymentResponseHeader  = "PAYMENT-RESPONSE"
)

// X402HTTPClient wraps http.Client with 402 payment handling
type X402HTTPClient struct {
	Client     *http.Client
	X402Client *X402Client
	Selector   func([]PaymentRequirements) *PaymentRequirements
}

// NewX402HTTPClient creates a new HTTP client with payment support
func NewX402HTTPClient(httpClient *http.Client, x402Client *X402Client) *X402HTTPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &X402HTTPClient{
		Client:     httpClient,
		X402Client: x402Client,
	}
}

// Get performs GET with 402 handling
func (c *X402HTTPClient) Get(ctx context.Context, urlStr string) (*http.Response, error) {
	return c.RequestWithPayment(ctx, "GET", urlStr, nil)
}

// Post performs POST with 402 handling
func (c *X402HTTPClient) Post(ctx context.Context, urlStr string, body io.Reader) (*http.Response, error) {
	return c.RequestWithPayment(ctx, "POST", urlStr, body)
}

// RequestWithPayment makes request and retries with payment on 402
func (c *X402HTTPClient) RequestWithPayment(ctx context.Context, method, urlStr string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 402 {
		return resp, nil
	}
	// Parse PaymentRequired
	paymentRequired := c.parsePaymentRequired(resp)
	if paymentRequired == nil {
		return resp, nil
	}
	// Create payment payload
	resource := urlStr
	extensions := make(map[string]any)
	if paymentRequired.Extensions != nil && paymentRequired.Extensions.Permit402Context != nil {
		extensions["permit402Context"] = map[string]any{
			"meta": map[string]any{
				"kind":        paymentRequired.Extensions.Permit402Context.Meta.Kind,
				"paymentId":   paymentRequired.Extensions.Permit402Context.Meta.PaymentID,
				"nonce":       paymentRequired.Extensions.Permit402Context.Meta.Nonce,
				"validAfter":  paymentRequired.Extensions.Permit402Context.Meta.ValidAfter,
				"validBefore": paymentRequired.Extensions.Permit402Context.Meta.ValidBefore,
			},
		}
	}
	var filters *PaymentRequirementsFilter
	if c.Selector != nil {
		sel := c.Selector(paymentRequired.Accepts)
		if sel != nil {
			filters = &PaymentRequirementsFilter{Scheme: sel.Scheme, Network: sel.Network}
		}
	}
	payload, err := c.X402Client.HandlePayment(ctx, paymentRequired.Accepts, resource, extensions, filters)
	if err != nil {
		return resp, err
	}
	// Retry with payment
	return c.retryWithPayment(ctx, method, urlStr, body, payload)
}

func (c *X402HTTPClient) parsePaymentRequired(resp *http.Response) *PaymentRequired {
	// Try header first
	if h := resp.Header.Get(PaymentRequiredHeader); h != "" {
		decoded, err := base64.StdEncoding.DecodeString(h)
		if err != nil {
			return nil
		}
		var pr PaymentRequired
		if json.Unmarshal(decoded, &pr) == nil && len(pr.Accepts) > 0 {
			return &pr
		}
	}
	// Try body
	dec := json.NewDecoder(resp.Body)
	var pr PaymentRequired
	if dec.Decode(&pr) == nil && len(pr.Accepts) > 0 {
		return &pr
	}
	return nil
}

func (c *X402HTTPClient) retryWithPayment(ctx context.Context, method, urlStr string, body io.Reader, payload *PaymentPayload) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	encoded, err := EncodePaymentPayload(payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set(PaymentSignatureHeader, encoded)
	return c.Client.Do(req)
}
