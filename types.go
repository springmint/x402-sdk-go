package x402

// Delivery ptype constants
const (
	PaymentOnly = "PAYMENT_ONLY"
)

// PtypeMap maps delivery ptype string to EIP-712 numeric value
var PtypeMap = map[string]uint8{
	PaymentOnly: 0,
}

// PermitMeta is payment permit metadata
type PermitMeta struct {
	Ptype      string `json:"ptype"`
	PaymentID  string `json:"paymentId"`
	Nonce      string `json:"nonce"`
	ValidAfter int64  `json:"validAfter"`
	ValidBefore int64 `json:"validBefore"`
}

// Payment is payment information
type Payment struct {
	PayToken  string `json:"payToken"`
	PayAmount string `json:"payAmount"`
	PayTo     string `json:"payTo"`
}

// Fee is fee information
type Fee struct {
	FeeTo     string `json:"feeTo"`
	FeeAmount string `json:"feeAmount"`
}

// Permit402 is the payment permit structure (matches contract Permit402Details)
type Permit402 struct {
	Meta    PermitMeta `json:"meta"`
	Buyer   string     `json:"buyer"`
	Payment Payment    `json:"payment"`
	Fee     Fee        `json:"fee"`
}

// FeeInfo is fee information in payment requirements
type FeeInfo struct {
	FacilitatorID string `json:"facilitatorId,omitempty"`
	FeeTo         string `json:"feeTo"`
	FeeAmount     string `json:"feeAmount"`
}

// PaymentRequirementsExtra is extra information in payment requirements
type PaymentRequirementsExtra struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Fee     *FeeInfo `json:"fee,omitempty"`
}

// PaymentRequirements is payment requirements from server
type PaymentRequirements struct {
	Scheme           string                    `json:"scheme"`
	Network          string                    `json:"network"`
	Amount           string                    `json:"amount"`
	Asset            string                    `json:"asset"`
	PayTo            string                    `json:"payTo"`
	MaxTimeoutSeconds *int                     `json:"maxTimeoutSeconds,omitempty"`
	Extra            *PaymentRequirementsExtra `json:"extra,omitempty"`
}

// Permit402ContextMeta is meta in permit402 context
type Permit402ContextMeta struct {
	Ptype      string `json:"ptype"`
	PaymentID  string `json:"paymentId"`
	Nonce      string `json:"nonce"`
	ValidAfter int64  `json:"validAfter"`
	ValidBefore int64 `json:"validBefore"`
}

// Permit402Context is permit402 context from extensions
type Permit402Context struct {
	Meta Permit402ContextMeta `json:"meta"`
}

// ResourceInfo is resource information
type ResourceInfo struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// PaymentRequiredExtensions is extensions in PaymentRequired
type PaymentRequiredExtensions struct {
	Permit402Context *Permit402Context `json:"permit402Context,omitempty"`
}

// PaymentRequired is 402 payment required response
type PaymentRequired struct {
	X402Version int                   `json:"x402Version"`
	Error       string                `json:"error,omitempty"`
	Resource    *ResourceInfo         `json:"resource,omitempty"`
	Accepts     []PaymentRequirements `json:"accepts"`
	Extensions  *PaymentRequiredExtensions `json:"extensions,omitempty"`
}

// PaymentPayloadData is payment payload data
type PaymentPayloadData struct {
	Signature string     `json:"signature"`
	Permit402 *Permit402 `json:"permit402,omitempty"`
}

// PaymentPayload is payment payload sent by client
type PaymentPayload struct {
	X402Version int                 `json:"x402Version"`
	Resource    *ResourceInfo       `json:"resource,omitempty"`
	Accepted    PaymentRequirements `json:"accepted"`
	Payload     PaymentPayloadData  `json:"payload"`
	Extensions  map[string]any      `json:"extensions,omitempty"`
}

// VerifyResponse is verification response from facilitator
type VerifyResponse struct {
	IsValid      bool   `json:"isValid"`
	InvalidReason string `json:"invalidReason,omitempty"`
}

// TransactionInfo is transaction information
type TransactionInfo struct {
	Hash        string `json:"hash"`
	BlockNumber string `json:"blockNumber,omitempty"`
	Status      string `json:"status,omitempty"`
}

// SettleResponse is settlement response from facilitator
type SettleResponse struct {
	Success     bool   `json:"success"`
	Transaction string `json:"transaction,omitempty"`
	Network     string `json:"network,omitempty"`
	ErrorReason string `json:"errorReason,omitempty"`
}

// SupportedPtype is supported payment ptype
type SupportedPtype struct {
	X402Version int    `json:"x402Version"`
	Scheme      string `json:"scheme"`
	Network     string `json:"network"`
}

// SupportedFee is supported fee configuration
type SupportedFee struct {
	FeeTo   string `json:"feeTo"`
	Pricing string `json:"pricing"` // "per_accept" or "flat"
}

// PaymentConfigItem is a payment config item returned by facilitator GET /payment/config
type PaymentConfigItem struct {
	Chain  string `json:"chain"`
	Token  string `json:"token"`
	Amount string `json:"amount"`
	PayTo  string `json:"payTo"`
}

// SupportedResponse is supported response from facilitator
type SupportedResponse struct {
	Ptypes []SupportedPtype `json:"ptypes"`
	Fee    SupportedFee     `json:"fee"`
}

// FeeQuoteResponse is fee quote response from facilitator
type FeeQuoteResponse struct {
	Fee      FeeInfo `json:"fee"`
	Pricing  string  `json:"pricing"`
	Scheme   string  `json:"scheme"`
	Network  string  `json:"network"`
	Asset    string  `json:"asset"`
	ExpiresAt *int64 `json:"expiresAt,omitempty"`
}
