package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"rapid/lib/errors"
	"rapid/services/download/rpc/api"
)

// JSONRPCVersion is the aria2 JSON-RPC version.
const JSONRPCVersion = "2.0"

// StatusKeys are the tellStatus fields fetched for status updates.
var StatusKeys = []string{
	"completedLength",
	"connections",
	"dir",
	"downloadSpeed",
	"errorCode",
	"errorMessage",
	"files",
	"gid",
	"numPieces",
	"pieceLength",
	"status",
	"totalLength",
	"verifiedLength",
}

// JSONRpc speaks aria2's HTTP JSON-RPC endpoint.
type JSONRpc struct {
	Host       string
	Port       int
	Token      string
	HTTPClient *http.Client
	// BaseURL overrides the endpoint in tests (httptest server).
	BaseURL string

	mu     sync.Mutex
	nextID int
}

func NewJSONRpc(host string, port int, token string, client ...*http.Client) *JSONRpc {
	httpClient := http.DefaultClient
	if len(client) > 0 && client[0] != nil {
		httpClient = client[0]
	}

	return &JSONRpc{
		Host:       host,
		Port:       port,
		Token:      token,
		HTTPClient: httpClient,
		BaseURL:    fmt.Sprintf("http://%s:%d/jsonrpc", host, port),
	}
}

func (r *JSONRpc) bodyAdapter(method string, body []byte) ([]byte, error) {
	r.mu.Lock()
	r.nextID++
	id := r.nextID
	r.mu.Unlock()

	var v []any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &v); err != nil {
			return nil, err
		}
	}

	rpcParams := make([]any, 0)
	if r.Token != "" {
		rpcParams = append(rpcParams, "token:"+r.Token)
	}
	rpcParams = append(rpcParams, v...)

	body, err := json.Marshal(map[string]any{
		"jsonrpc": JSONRPCVersion,
		"id":      id,
		"method":  method,
		"params":  rpcParams,
	})
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (r *JSONRpc) Call(ctx context.Context, method string, body []byte) ([]byte, error) {
	body, err := r.bodyAdapter(method, body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Aria2Errorf("cannot connect to aria2: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Aria2Errorf("cannot connect to aria2: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Aria2Errorf("cannot connect to aria2: %v", err)
	}

	var v api.JSONRpcResponse
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, errors.Aria2Errorf("cannot unmarshal response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || v.Error != nil {
		if v.Error != nil {
			return nil, errors.Aria2Errorf("aria2 error code %d: %s", v.Error.Code, v.Error.Message)
		}

		return nil, errors.Aria2Errorf("aria2 HTTP error %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return data, nil
}
