// Package goframemw provides x402 payment middleware for the GoFrame framework.
//
// Usage:
//
//	import (
//	    x402 "github.com/springmint/x402-sdk-go"
//	    "github.com/springmint/x402-sdk-go/middleware/goframemw"
//	)
//
//	// Protect a single route:
//	s.BindMiddleware("/protected", goframemw.Paywall(server, "eip155:97", "0.0001 USDC", "0xPayTo..."))
//
//	// Protect a group:
//	s.Group("/api", func(group *ghttp.RouterGroup) {
//	    group.Middleware(goframemw.Paywall(server, "eip155:97", "0.0001 USDC", "0xPayTo..."))
//	    group.GET("/resource", handler)
//	})
package goframemw

import (
	"github.com/gogf/gf/v2/net/ghttp"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/middleware"
)

const (
	ctxKeySettlement = "x402_settlement"
	ctxKeyAccepted   = "x402_accepted"
)

// Paywall returns a GoFrame middleware that protects an endpoint with x402 payment.
//
//	s.BindMiddleware("/protected", goframemw.Paywall(server, "eip155:97", "0.0001 USDC", "0xPayTo..."))
func Paywall(server *x402.X402Server, network, price, payTo string) ghttp.HandlerFunc {
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

// PaywallWithConfig returns a GoFrame middleware with full configuration control.
func PaywallWithConfig(cfg middleware.PaywallConfig) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		rw := &gfResponseWriter{r: r}
		result := middleware.ProcessPayment(r.Context(), &cfg, r.Request, rw)
		if result == nil {
			return // 402 or error already written
		}
		r.SetCtxVar(ctxKeySettlement, result.SettleResp)
		r.SetCtxVar(ctxKeyAccepted, &result.Accepted)
		r.Middleware.Next()
	}
}

// PaywallWithPriceFunc returns a GoFrame middleware with dynamic pricing per request.
//
//	s.BindMiddleware("/ai/generate", goframemw.PaywallWithPriceFunc(server, "eip155:97", "0xPayTo...",
//	    func(r *ghttp.Request) string {
//	        if r.Get("model").String() == "gpt-4" { return "0.01 USDC" }
//	        return "0.001 USDC"
//	    },
//	))
func PaywallWithPriceFunc(server *x402.X402Server, network, payTo string, priceFunc func(*ghttp.Request) string) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		price := priceFunc(r)
		cfg := middleware.PaywallConfig{
			Server: server,
			Resource: &x402.ResourceConfig{
				Scheme:  "permit402",
				Network: network,
				Price:   price,
				PayTo:   payTo,
			},
		}
		rw := &gfResponseWriter{r: r}
		result := middleware.ProcessPayment(r.Context(), &cfg, r.Request, rw)
		if result == nil {
			return
		}
		r.SetCtxVar(ctxKeySettlement, result.SettleResp)
		r.SetCtxVar(ctxKeyAccepted, &result.Accepted)
		r.Middleware.Next()
	}
}

// GetSettlement retrieves the settlement response from GoFrame request context.
func GetSettlement(r *ghttp.Request) *x402.SettleResponse {
	if v := r.GetCtxVar(ctxKeySettlement); !v.IsNil() {
		if s, ok := v.Val().(*x402.SettleResponse); ok {
			return s
		}
	}
	return nil
}

// GetAccepted retrieves the accepted payment requirements from GoFrame request context.
func GetAccepted(r *ghttp.Request) *x402.PaymentRequirements {
	if v := r.GetCtxVar(ctxKeyAccepted); !v.IsNil() {
		if s, ok := v.Val().(*x402.PaymentRequirements); ok {
			return s
		}
	}
	return nil
}

type gfResponseWriter struct {
	r *ghttp.Request
}

func (g *gfResponseWriter) SetHeader(key, value string) {
	g.r.Response.Header().Set(key, value)
}

func (g *gfResponseWriter) WriteError(status int, msg string) {
	g.r.Response.WriteStatus(status, msg)
}

func (g *gfResponseWriter) WriteResponse(status int, body []byte) {
	// Pass body to WriteStatus to avoid GoFrame writing default status text (e.g. "Payment Required")
	// which would corrupt the JSON body
	g.r.Response.WriteStatus(status, body)
}
