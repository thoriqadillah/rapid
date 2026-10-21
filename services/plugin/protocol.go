package plugin

// Line-delimited JSON-RPC 2.0 over the child's stdin/stdout, one object per
// line. Params is always a list: real plugins read positional params
// (params[0] is the URI for rapid.resolve).
const (
	JSONRPCVersion = "2.0"
	MethodPing     = "rapid.ping"
	MethodResolve  = "rapid.resolve"
)

// Request is one outgoing call.
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

// PingRequest builds a rapid.ping health probe.
func PingRequest(id int) Request {
	return Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodPing,
		Params:  []any{},
	}
}

// ResolveRequest builds a rapid.resolve call with params [uri, options].
func ResolveRequest(id int, uri string, options map[string]any) Request {
	if options == nil {
		options = map[string]any{}
	}
	return Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodResolve,
		Params:  []any{uri, options},
	}
}

// Response is one reply line.
type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Error   *ResponseError `json:"error,omitempty"`
}

// ResponseError is a JSON-RPC error object.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// PingResult is a rapid.ping answer.
type PingResult struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Type    string   `json:"type"`
	Schemes []string `json:"schemes"`
}

type PingResponse struct {
	Response
	Result PingResult `json:"result"`
}

// ResolvedItem is one rapid.resolve result entry.
type ResolvedItem struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	Category string `json:"category,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

// ResolveResult wraps the items list real plugins send under result.
type ResolveResult struct {
	Items []ResolvedItem `json:"items"`
}

type ResolveResponse struct {
	Response
	Result *ResolveResult `json:"result"`
}
