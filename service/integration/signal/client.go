package signal

import (
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
)

var rpcID int32

type Client struct {
	SocketPath string
}

func NewClient(socketPath string) *Client {
	return &Client{
		SocketPath: socketPath,
	}
}

type RPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int32          `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
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

func (c *Client) Call(method string, params map[string]any) (json.RawMessage, error) {
	conn, err := net.Dial("unix", c.SocketPath)
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

func (c *Client) CallWithAccount(method string, params map[string]any, account string) (json.RawMessage, error) {
	if params == nil {
		params = make(map[string]any)
	}
	params["account"] = account
	return c.Call(method, params)
}

type AccountInfo struct {
	Number string `json:"number"`
	Name   string `json:"name,omitempty"`
}

type Account struct {
	Number string
}

func (c *Client) GetLinkedAccount() (*Account, error) {
	if c == nil {
		return nil, nil
	}

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

	return &Account{Number: accounts[0].Number}, nil
}

func (c *Client) CreateGroup(name string) (string, string, error) {
	if c == nil {
		return "", "", fmt.Errorf("signal client not initialized")
	}

	account, err := c.GetLinkedAccount()
	if err != nil {
		return "", "", fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		return "", "", fmt.Errorf("no linked Signal account")
	}

	params := map[string]any{
		"name":   name,
		"member": []string{},
	}

	result, err := c.CallWithAccount("updateGroup", params, account.Number)
	if err != nil {
		return "", "", err
	}

	var response struct {
		GroupID string `json:"groupId"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", "", fmt.Errorf("failed to parse updateGroup response: %w", err)
	}

	if response.GroupID == "" {
		return "", "", fmt.Errorf("empty groupId in response")
	}

	return response.GroupID, account.Number, nil
}

func (c *Client) SendGroupMessage(groupID, message string) error {
	if c == nil {
		return fmt.Errorf("signal client not initialized")
	}

	account, err := c.GetLinkedAccount()
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("no linked account")
	}

	params := map[string]any{
		"groupId":    groupID,
		"message":    message,
		"notifySelf": true,
	}

	_, err = c.CallWithAccount("send", params, account.Number)
	return err
}
