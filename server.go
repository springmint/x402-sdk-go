package x402

import (
	"context"
	"fmt"
	"time"

	"github.com/springmint/x402-sdk-go/tokens"
)

// ResourceConfig is resource payment configuration
type ResourceConfig struct {
	Scheme       string
	Network      string
	Price        string
	PayTo        string
	ValidFor     int
	DeliveryMode string
}

// ServerMechanism parses prices and validates requirements
type ServerMechanism interface {
	Scheme() string
	ParsePrice(price, network string) (amount, asset string, err error)
	ValidatePaymentRequirements(requirements PaymentRequirements) bool
}

// X402Server is the core payment server
type X402Server struct {
	mechanisms  map[string]map[string]ServerMechanism // network -> scheme -> mechanism
	facilitator FacilitatorClient
}

// NewX402Server creates a new X402Server
func NewX402Server() *X402Server {
	return &X402Server{
		mechanisms: make(map[string]map[string]ServerMechanism),
	}
}

// Register registers a mechanism for network
func (s *X402Server) Register(network string, mechanism ServerMechanism) *X402Server {
	if s.mechanisms[network] == nil {
		s.mechanisms[network] = make(map[string]ServerMechanism)
	}
	s.mechanisms[network][mechanism.Scheme()] = mechanism
	return s
}

// SetFacilitator sets the facilitator client
func (s *X402Server) SetFacilitator(client FacilitatorClient) *X402Server {
	s.facilitator = client
	return s
}

// BuildPaymentRequirementsFromAPIKey fetches payment config (chain/token/amount/payTo) from
// facilitator via GetPaymentConfig using the given apiKey and priceUSD, then builds PaymentRequirements.
func (s *X402Server) BuildPaymentRequirementsFromAPIKey(ctx context.Context, apiKey string, priceUSD string, scheme string, validFor int) ([]PaymentRequirements, error) {
	if s.facilitator == nil {
		return nil, &X402Error{Msg: "no facilitator configured"}
	}
	if apiKey == "" {
		return nil, &X402Error{Msg: "apiKey required"}
	}
	configs, err := s.facilitator.GetPaymentConfig(ctx, apiKey, priceUSD)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, nil
	}
	if scheme == "" {
		scheme = "transfer_auth"
	}
	if validFor <= 0 {
		validFor = 3600
	}

	var result []PaymentRequirements
	for _, cfg := range configs {
		mech := s.findMechanism(cfg.Chain, scheme)
		if mech == nil {
			continue
		}
		asset := ""
		if ti := tokens.GetToken(cfg.Chain, cfg.Token); ti != nil {
			asset = ti.Address
		}
		if asset == "" {
			continue
		}
		amount := cfg.Amount
		if amount == "" {
			// skip if facilitator did not return amount (priceUSD not provided)
			continue
		}
		req := PaymentRequirements{
			Scheme:            scheme,
			Network:           cfg.Chain,
			Amount:            amount,
			Asset:             asset,
			PayTo:             cfg.PayTo,
			MaxTimeoutSeconds: &validFor,
		}
		result = append(result, req)
	}
	if len(result) == 0 {
		return nil, nil
	}
	if s.facilitator != nil {
		feeQuotes, err := s.facilitator.FeeQuote(ctx, result)
		if err == nil {
			for i := range result {
				if i < len(feeQuotes) && feeQuotes[i] != nil {
					r := &result[i]
					if r.Extra == nil {
						r.Extra = &PaymentRequirementsExtra{}
					}
					r.Extra.Fee = &feeQuotes[i].Fee
				}
			}
		}
	}
	return result, nil
}

// BuildPaymentRequirements builds requirements from configs
func (s *X402Server) BuildPaymentRequirements(ctx context.Context, configs []ResourceConfig) ([]PaymentRequirements, error) {

	var result []PaymentRequirements
	for _, cfg := range configs {
		mech := s.findMechanism(cfg.Network, cfg.Scheme)
		if mech == nil {
			return nil, &X402Error{Msg: "no mechanism for " + cfg.Network + "/" + cfg.Scheme}
		}
		amount, asset, err := mech.ParsePrice(cfg.Price, cfg.Network)
		if err != nil {
			return nil, err
		}
		req := PaymentRequirements{
			Scheme:            cfg.Scheme,
			Network:           cfg.Network,
			Amount:            amount,
			Asset:             asset,
			PayTo:             cfg.PayTo,
			MaxTimeoutSeconds: intPtr(cfg.ValidFor),
		}
		if req.MaxTimeoutSeconds == nil || *req.MaxTimeoutSeconds == 0 {
			validFor := 3600
			req.MaxTimeoutSeconds = &validFor
		}
		result = append(result, req)
	}
	if s.facilitator != nil {
		feeQuotes, err := s.facilitator.FeeQuote(ctx, result)
		if err == nil {
			for i := range result {
				if i < len(feeQuotes) && feeQuotes[i] != nil {
					req := &result[i]
					if req.Extra == nil {
						req.Extra = &PaymentRequirementsExtra{}
					}
					req.Extra.Fee = &feeQuotes[i].Fee
				}
			}
		}
	}
	return result, nil
}

// CreatePaymentRequiredResponse creates 402 response
func (s *X402Server) CreatePaymentRequiredResponse(requirements []PaymentRequirements, paymentID, nonce string, validAfter, validBefore *int64) *PaymentRequired {
	now := time.Now().Unix()
	if validAfter == nil {
		v := now
		validAfter = &v
	}
	if validBefore == nil {
		v := now + 3600
		validBefore = &v
	}
	if paymentID == "" {
		paymentID = GeneratePaymentID()
	}
	if nonce == "" {
		// generate unique nonce to avoid Permit402.NonceAlreadyUsed
		nonce = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	ext := &PaymentRequiredExtensions{
		Permit402Context: &Permit402Context{
			Meta: Permit402ContextMeta{
				Kind:        PaymentOnly,
				PaymentID:   paymentID,
				Nonce:       nonce,
				ValidAfter:  *validAfter,
				ValidBefore: *validBefore,
			},
		},
	}
	return &PaymentRequired{
		X402Version: 2,
		Error:       "Payment required",
		Accepts:     requirements,
		Extensions:  ext,
	}
}

// VerifyPayment verifies payload
func (s *X402Server) VerifyPayment(ctx context.Context, payload *PaymentPayload, requirements PaymentRequirements) (*VerifyResponse, error) {
	if s.facilitator == nil {
		return &VerifyResponse{IsValid: false, InvalidReason: "no_facilitator"}, nil
	}
	return s.facilitator.Verify(ctx, payload, requirements)
}

// SettlePayment settles payment
func (s *X402Server) SettlePayment(ctx context.Context, payload *PaymentPayload, requirements PaymentRequirements) (*SettleResponse, error) {
	if s.facilitator == nil {
		return &SettleResponse{Success: false, ErrorReason: "no_facilitator"}, nil
	}
	return s.facilitator.Settle(ctx, payload, requirements)
}

func (s *X402Server) findMechanism(network, scheme string) ServerMechanism {
	if m, ok := s.mechanisms[network]; ok {
		return m[scheme]
	}
	return nil
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}
