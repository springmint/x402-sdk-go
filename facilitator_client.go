package x402

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPFacilitatorClient implements FacilitatorClient by calling the facilitator HTTP API.
type HTTPFacilitatorClient struct {
	BaseURL       string
	Headers       map[string]string
	FacilitatorID string
	Client        *http.Client
}

// HTTPFacilitatorClientOption configures HTTPFacilitatorClient
type HTTPFacilitatorClientOption func(*HTTPFacilitatorClient)

// WithFacilitatorHTTPClient sets a custom *http.Client
func WithFacilitatorHTTPClient(c *http.Client) HTTPFacilitatorClientOption {
	return func(f *HTTPFacilitatorClient) { f.Client = c }
}

// WithFacilitatorHeaders sets custom headers (e.g. Authorization)
func WithFacilitatorHeaders(h map[string]string) HTTPFacilitatorClientOption {
	return func(f *HTTPFacilitatorClient) { f.Headers = h }
}

// WithFacilitatorID sets the facilitator identifier
func WithFacilitatorID(id string) HTTPFacilitatorClientOption {
	return func(f *HTTPFacilitatorClient) { f.FacilitatorID = id }
}

// NewHTTPFacilitatorClient creates a client for the facilitator service at baseURL.
func NewHTTPFacilitatorClient(baseURL string, opts ...HTTPFacilitatorClientOption) *HTTPFacilitatorClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	c := &HTTPFacilitatorClient{
		BaseURL:       baseURL,
		Headers:       make(map[string]string),
		FacilitatorID: baseURL,
		Client:        &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FeeQuote calls POST /fee/quote
func (c *HTTPFacilitatorClient) FeeQuote(ctx context.Context, accepts []PaymentRequirements) ([]*FeeQuoteResponse, error) {
	body := struct {
		Accepts []PaymentRequirements `json:"accepts"`
	}{Accepts: accepts}
	var raw []*FeeQuoteResponse
	if err := c.postJSON(ctx, "/fee/quote", body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// Verify calls POST /verify
func (c *HTTPFacilitatorClient) Verify(ctx context.Context, payload *PaymentPayload, requirements PaymentRequirements) (*VerifyResponse, error) {
	body := struct {
		PaymentPayload       *PaymentPayload       `json:"paymentPayload"`
		PaymentRequirements PaymentRequirements   `json:"paymentRequirements"`
	}{payload, requirements}
	var out VerifyResponse
	if err := c.postJSON(ctx, "/verify", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Settle calls POST /settle
func (c *HTTPFacilitatorClient) Settle(ctx context.Context, payload *PaymentPayload, requirements PaymentRequirements) (*SettleResponse, error) {
	body := struct {
		PaymentPayload       *PaymentPayload     `json:"paymentPayload"`
		PaymentRequirements  PaymentRequirements `json:"paymentRequirements"`
	}{payload, requirements}
	var out SettleResponse
	if err := c.postJSON(ctx, "/settle", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPaymentConfig calls GET /payment/config with X-API-KEY header
func (c *HTTPFacilitatorClient) GetPaymentConfig(ctx context.Context, apiKey string, priceUSD string) ([]PaymentConfigItem, error) {
	path := "/payment/config"
	if priceUSD != "" {
		path = path + "?price=" + url.QueryEscape(priceUSD)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	if apiKey != "" {
		req.Header.Set("X-API-KEY", apiKey)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("facilitator %s: %s", path, resp.Status)
	}
	var out struct {
		Configs []PaymentConfigItem `json:"configs"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out.Configs, nil
}

// Supported calls GET /supported (optional, not part of FacilitatorClient interface)
func (c *HTTPFacilitatorClient) Supported(ctx context.Context) (*SupportedResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/supported", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facilitator supported: %s", resp.Status)
	}
	var out SupportedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *HTTPFacilitatorClient) postJSON(ctx context.Context, path string, body any, result any) error {
	u := c.BaseURL + path
	enc, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(enc))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("facilitator %s: %s", path, resp.Status)
	}
	return json.Unmarshal(data, result)
}

func (c *HTTPFacilitatorClient) setHeaders(req *http.Request) {
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
}

// Ensure HTTPFacilitatorClient implements FacilitatorClient
var _ FacilitatorClient = (*HTTPFacilitatorClient)(nil)
