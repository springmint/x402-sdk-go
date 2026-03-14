package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	x402 "github.com/springmint/x402-sdk-go"
)

// PaywallConfig defines the payment requirements for a protected endpoint.
type PaywallConfig struct {
	// Server is the initialized X402Server instance (with mechanisms registered and facilitator set).
	Server *x402.X402Server

	// Resource defines the payment requirements for this endpoint.
	// Either Resource or APIKeyHeader must be set.
	Resource *x402.ResourceConfig

	// APIKeyHeader enables dynamic pricing via facilitator's GetPaymentConfig.
	// When set, the middleware reads the API key from this request header (e.g. "X-API-KEY")
	// and calls BuildPaymentRequirementsFromAPIKey. Falls back to Resource if no key is present.
	APIKeyHeader string

	// Scheme is used with APIKeyHeader mode. Defaults to "permit402".
	Scheme string

	// ValidFor is the payment timeout in seconds. Defaults to 3600.
	ValidFor int
}

func (c *PaywallConfig) defaults() {
	if c.Scheme == "" {
		c.Scheme = "permit402"
	}
	if c.ValidFor <= 0 {
		c.ValidFor = 3600
	}
}

// PaywallResult holds the result of a successful payment processing.
type PaywallResult struct {
	SettleResp *x402.SettleResponse
	Accepted   x402.PaymentRequirements
}

// ResponseWriter abstracts the response writing for different frameworks.
type ResponseWriter interface {
	SetHeader(key, value string)
	WriteError(status int, msg string)
	WriteResponse(status int, body []byte)
}

// ProcessPayment handles the core x402 payment flow.
// Returns nil if a 402 or error response was written to w; returns PaywallResult on success.
func ProcessPayment(ctx context.Context, cfg *PaywallConfig, r *http.Request, w ResponseWriter) *PaywallResult {
	cfg.defaults()

	// Prevent caching of paid responses — avoid replay without payment
	w.SetHeader("Cache-Control", "no-store")

	// Check if client already provided payment
	sig := r.Header.Get(x402.PaymentSignatureHeader)
	if sig != "" {
		return verifyAndSettle(ctx, cfg, sig, w)
	}

	// No payment - return 402
	return returnPaymentRequired(ctx, cfg, r, w)
}

func verifyAndSettle(ctx context.Context, cfg *PaywallConfig, sig string, w ResponseWriter) *PaywallResult {
	payload, err := x402.DecodePaymentPayload[x402.PaymentPayload](sig)
	if err != nil {
		w.WriteError(http.StatusBadRequest, "invalid payment payload: "+err.Error())
		return nil
	}

	req := payload.Accepted

	verifyResp, err := cfg.Server.VerifyPayment(ctx, payload, req)
	if err != nil {
		w.WriteError(http.StatusBadRequest, "verify error: "+err.Error())
		return nil
	}
	if !verifyResp.IsValid {
		w.WriteError(http.StatusPaymentRequired, "invalid payment: "+verifyResp.InvalidReason)
		return nil
	}

	settleResp, err := cfg.Server.SettlePayment(ctx, payload, req)
	if err != nil {
		w.WriteError(http.StatusBadRequest, "settle error: "+err.Error())
		return nil
	}
	if !settleResp.Success {
		w.WriteError(http.StatusBadRequest, "settle failed: "+settleResp.ErrorReason)
		return nil
	}

	// Set payment response header
	if encoded, err := x402.EncodePaymentPayload(settleResp); err == nil {
		w.SetHeader(x402.PaymentResponseHeader, encoded)
	}

	return &PaywallResult{SettleResp: settleResp, Accepted: req}
}

func returnPaymentRequired(ctx context.Context, cfg *PaywallConfig, r *http.Request, w ResponseWriter) *PaywallResult {
	requirements, err := buildRequirements(ctx, cfg, r)
	if err != nil {
		w.WriteError(http.StatusInternalServerError, "failed to build payment requirements: "+err.Error())
		return nil
	}

	pr := cfg.Server.CreatePaymentRequiredResponse(requirements, "", "", nil, nil)
	body, err := json.Marshal(pr)
	if err != nil {
		w.WriteError(http.StatusInternalServerError, "failed to marshal payment required: "+err.Error())
		return nil
	}

	w.SetHeader("Content-Type", "application/json")
	w.SetHeader(x402.PaymentRequiredHeader, x402.EncodeBase64(body))
	w.WriteResponse(http.StatusPaymentRequired, body)
	return nil
}

func buildRequirements(ctx context.Context, cfg *PaywallConfig, r *http.Request) ([]x402.PaymentRequirements, error) {
	// Try API key mode first
	if cfg.APIKeyHeader != "" {
		if apiKey := r.Header.Get(cfg.APIKeyHeader); apiKey != "" {
			price := ""
			if cfg.Resource != nil {
				price = cfg.Resource.Price
			}
			reqs, err := cfg.Server.BuildPaymentRequirementsFromAPIKey(ctx, apiKey, price, cfg.Scheme, cfg.ValidFor)
			if err != nil {
				return nil, err
			}
			if len(reqs) > 0 {
				return reqs, nil
			}
		}
	}

	// Fall back to static resource config
	if cfg.Resource == nil {
		return nil, &x402.X402Error{Msg: "no resource config and no API key"}
	}
	rc := *cfg.Resource
	if rc.ValidFor <= 0 {
		rc.ValidFor = cfg.ValidFor
	}
	return cfg.Server.BuildPaymentRequirements(ctx, []x402.ResourceConfig{rc})
}

// --- net/http middleware ---

// Paywall returns a net/http middleware that protects an endpoint with x402 payment.
//
// Usage:
//
//	mux.Handle("/protected", middleware.Paywall(cfg, http.HandlerFunc(myHandler)))
func Paywall(cfg PaywallConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &stdResponseWriter{w: w}
		result := ProcessPayment(r.Context(), &cfg, r, rw)
		if result == nil {
			return // 402 or error already sent
		}
		ctx := context.WithValue(r.Context(), ctxKeyPaywallResult, result)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PaywallFunc is a convenience wrapper for http.HandlerFunc.
func PaywallFunc(cfg PaywallConfig, handler http.HandlerFunc) http.Handler {
	return Paywall(cfg, handler)
}

// PaywallMiddleware returns a middleware function compatible with common routers (chi, etc.)
// that accept func(http.Handler) http.Handler.
//
// Usage with chi:
//
//	r.With(middleware.PaywallMiddleware(cfg)).Get("/protected", myHandler)
func PaywallMiddleware(cfg PaywallConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Paywall(cfg, next)
	}
}

// GetSettlement retrieves the settlement response from the request context.
func GetSettlement(ctx context.Context) *x402.SettleResponse {
	if r, ok := ctx.Value(ctxKeyPaywallResult).(*PaywallResult); ok {
		return r.SettleResp
	}
	return nil
}

// GetAccepted retrieves the accepted payment requirements from the request context.
func GetAccepted(ctx context.Context) *x402.PaymentRequirements {
	if r, ok := ctx.Value(ctxKeyPaywallResult).(*PaywallResult); ok {
		return &r.Accepted
	}
	return nil
}

type contextKey int

const ctxKeyPaywallResult contextKey = iota

type stdResponseWriter struct {
	w http.ResponseWriter
}

func (s *stdResponseWriter) SetHeader(key, value string) {
	s.w.Header().Set(key, value)
}

func (s *stdResponseWriter) WriteError(status int, msg string) {
	http.Error(s.w, msg, status)
}

func (s *stdResponseWriter) WriteResponse(status int, body []byte) {
	s.w.WriteHeader(status)
	_, _ = s.w.Write(body)
}
