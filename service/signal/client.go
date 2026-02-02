package signal

import (
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
)

var rpcID int32

type Client struct {
	socketPath string
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

type RPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int32                  `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int32           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Call(method string, params map[string]interface{}) (json.RawMessage, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      atomic.AddInt32(&rpcID, 1),
		Method:  method,
		Params:  params,
	}

	if err := json.NewEncoder(conn).Encode(&request); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	var response RPCResponse
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("signal-cli error: %s", response.Error.Message)
	}

	return response.Result, nil
}

func (c *Client) CallWithAccount(method string, params map[string]interface{}, account string) (json.RawMessage, error) {
	if params == nil {
		params = make(map[string]interface{})
	}
	params["account"] = account
	return c.Call(method, params)
}

type AccountInfo struct {
	Number string `json:"number"`
	Name   string `json:"name,omitempty"`
}

func (c *Client) GetLinkedAccount() (*AccountInfo, error) {
	result, err := c.Call("listAccounts", nil)
	if err != nil {
		return nil, err
	}

	var accounts []AccountInfo
	if err := json.Unmarshal(result, &accounts); err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, nil
	}

	return &accounts[0], nil
}
