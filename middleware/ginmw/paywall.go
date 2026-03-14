// Package ginmw provides x402 payment middleware for the Gin framework.
//
// Usage:
//
//	import (
//	    x402 "github.com/springmint/x402-sdk-go"
//	    "github.com/springmint/x402-sdk-go/middleware/ginmw"
//	)
//
//	// One line to protect a route:
//	r.GET("/protected", ginmw.Paywall(server, "eip155:97", "0.0001 USDC", "0xPayTo..."), myHandler)
//
//	// Or protect an entire group:
//	api := r.Group("/api", ginmw.Paywall(server, "eip155:97", "0.0001 USDC", "0xPayTo..."))
//	api.GET("/resource1", handler1)
//	api.GET("/resource2", handler2)
package ginmw

import (
	"github.com/gin-gonic/gin"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/middleware"
)

const (
	ctxKeySettlement = "x402_settlement"
	ctxKeyAccepted   = "x402_accepted"
)

// Paywall returns a Gin middleware that protects an endpoint with x402 payment.
// This is the simplest way to add payment to a route - just one line.
//
//	r.GET("/protected", ginmw.Paywall(server, "eip155:97", "0.0001 USDC", "0xPayTo..."), handler)
func Paywall(server *x402.X402Server, network, price, payTo string) gin.HandlerFunc {
	return PaywallWithConfig(middleware.PaywallConfig{
		Server: server,
		Resource: &x402.ResourceConfig{
			Scheme:  "permit402",
			Network: network,
			Price:   price,
			PayTo:   payTo,
		},
	})
}

// PaywallWithConfig returns a Gin middleware with full configuration control.
//
//	r.GET("/protected", ginmw.PaywallWithConfig(middleware.PaywallConfig{
//	    Server:       server,
//	    Resource:     &x402.ResourceConfig{...},
//	    APIKeyHeader: "X-API-KEY",
//	}), handler)
func PaywallWithConfig(cfg middleware.PaywallConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		rw := &ginResponseWriter{c: c}
		result := middleware.ProcessPayment(c.Request.Context(), &cfg, c.Request, rw)
		if result == nil {
			c.Abort()
			return
		}
		c.Set(ctxKeySettlement, result.SettleResp)
		c.Set(ctxKeyAccepted, &result.Accepted)
		c.Next()
	}
}

// PaywallWithPriceFunc returns a Gin middleware with dynamic pricing per request.
//
//	r.POST("/ai/generate", ginmw.PaywallWithPriceFunc(server, "eip155:97", "0xPayTo...",
//	    func(c *gin.Context) string {
//	        if c.Query("model") == "gpt-4" { return "0.01 USDC" }
//	        return "0.001 USDC"
//	    },
//	), handler)
func PaywallWithPriceFunc(server *x402.X402Server, network, payTo string, priceFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		price := priceFunc(c)
		cfg := middleware.PaywallConfig{
			Server: server,
			Resource: &x402.ResourceConfig{
				Scheme:  "permit402",
				Network: network,
				Price:   price,
				PayTo:   payTo,
			},
		}
		rw := &ginResponseWriter{c: c}
		result := middleware.ProcessPayment(c.Request.Context(), &cfg, c.Request, rw)
		if result == nil {
			c.Abort()
			return
		}
		c.Set(ctxKeySettlement, result.SettleResp)
		c.Set(ctxKeyAccepted, &result.Accepted)
		c.Next()
	}
}

// GetSettlement retrieves the settlement response from gin.Context.
func GetSettlement(c *gin.Context) *x402.SettleResponse {
	if v, ok := c.Get(ctxKeySettlement); ok {
		return v.(*x402.SettleResponse)
	}
	return nil
}

// GetAccepted retrieves the accepted payment requirements from gin.Context.
func GetAccepted(c *gin.Context) *x402.PaymentRequirements {
	if v, ok := c.Get(ctxKeyAccepted); ok {
		return v.(*x402.PaymentRequirements)
	}
	return nil
}

type ginResponseWriter struct {
	c *gin.Context
}

func (g *ginResponseWriter) SetHeader(key, value string) {
	g.c.Header(key, value)
}

func (g *ginResponseWriter) WriteError(status int, msg string) {
	g.c.String(status, msg)
}

func (g *ginResponseWriter) WriteResponse(status int, body []byte) {
	g.c.Data(status, "application/json", body)
}
