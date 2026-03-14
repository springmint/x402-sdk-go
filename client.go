package x402

import (
	"context"
	"strings"
)

// ClientMechanism creates payment payloads for a scheme (implemented by mechanisms package)
type ClientMechanism interface {
	Scheme() string
	CreatePaymentPayload(ctx context.Context, requirements PaymentRequirements, resource string, extensions map[string]any) (*PaymentPayload, error)
}

// PaymentRequirementsFilter filters requirements by scheme/network
type PaymentRequirementsFilter struct {
	Scheme  string
	Network string
}

// X402Client is the core payment client
type X402Client struct {
	mechanisms []struct {
		pattern   string
		mechanism ClientMechanism
		priority  int
	}
}

// NewX402Client creates a new X402Client
func NewX402Client() *X402Client {
	return &X402Client{}
}

// Register registers a mechanism for a network pattern (e.g. "eip155:*", "tron:shasta")
func (c *X402Client) Register(networkPattern string, mechanism ClientMechanism) *X402Client {
	priority := 1
	if !strings.HasSuffix(networkPattern, ":*") {
		priority = 10
	}
	c.mechanisms = append(c.mechanisms, struct {
		pattern   string
		mechanism ClientMechanism
		priority  int
	}{networkPattern, mechanism, priority})
	return c
}

// SelectPaymentRequirements selects from accepts using filters
func (c *X402Client) SelectPaymentRequirements(ctx context.Context, accepts []PaymentRequirements, filters *PaymentRequirementsFilter) (*PaymentRequirements, error) {
	candidates := accepts
	if filters != nil {
		if filters.Scheme != "" {
			var next []PaymentRequirements
			for _, r := range candidates {
				if r.Scheme == filters.Scheme {
					next = append(next, r)
				}
			}
			candidates = next
		}
		if filters.Network != "" {
			var next []PaymentRequirements
			for _, r := range candidates {
				if r.Network == filters.Network {
					next = append(next, r)
				}
			}
			candidates = next
		}
	}
	// Filter by mechanism support
	var supported []PaymentRequirements
	for _, r := range candidates {
		if c.findMechanism(r.Scheme, r.Network) != nil {
			supported = append(supported, r)
		}
	}
	if len(supported) == 0 {
		return nil, NewUnsupportedNetworkError("no supported payment requirements")
	}
	// Return first (or use token selection strategy)
	return &supported[0], nil
}

// CreatePaymentPayload creates payload for given requirements
func (c *X402Client) CreatePaymentPayload(ctx context.Context, requirements PaymentRequirements, resource string, extensions map[string]any) (*PaymentPayload, error) {
	mech := c.findMechanism(requirements.Scheme, requirements.Network)
	if mech == nil {
		return nil, NewUnsupportedNetworkError("no mechanism for " + requirements.Scheme + "/" + requirements.Network)
	}
	return mech.CreatePaymentPayload(ctx, requirements, resource, extensions)
}

// HandlePayment selects requirements and creates payload
func (c *X402Client) HandlePayment(ctx context.Context, accepts []PaymentRequirements, resource string, extensions map[string]any, filters *PaymentRequirementsFilter) (*PaymentPayload, error) {
	requirements, err := c.SelectPaymentRequirements(ctx, accepts, filters)
	if err != nil {
		return nil, err
	}
	return c.CreatePaymentPayload(ctx, *requirements, resource, extensions)
}

func (c *X402Client) findMechanism(scheme, network string) ClientMechanism {
	for _, e := range c.mechanisms {
		if e.mechanism.Scheme() != scheme {
			continue
		}
		if matchPattern(e.pattern, network) {
			return e.mechanism
		}
	}
	return nil
}

func matchPattern(pattern, network string) bool {
	if pattern == network {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(network, prefix)
	}
	return false
}
