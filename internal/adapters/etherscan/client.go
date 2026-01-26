package etherscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client handles communication with Etherscan API
type Client struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// NewClient creates a new Etherscan API client
func NewClient(client *http.Client, baseURL string, apiKey string) *Client {
	return &Client{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// APIResponse represents the standard Etherscan API response structure
type APIResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// Get makes a GET request to Etherscan API
func (c *Client) Get(ctx context.Context, params map[string]string, out interface{}) error {
	// Build query parameters
	query := url.Values{}
	query.Set("apikey", c.apiKey)
	for k, v := range params {
		query.Set(k, v)
	}
	query.Set("chainid", "1")

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, query.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Etherscan API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	// Etherscan returns status "1" for success, "0" for error
	if apiResp.Status == "0" {
		// Check if it's a valid "no results" case
		if apiResp.Message == "No transactions found" ||
			apiResp.Message == "No token transfers found" ||
			apiResp.Message == "No internal transactions found" {
			// Return empty result by unmarshaling empty array
			return json.Unmarshal([]byte("[]"), out)
		}
		// Otherwise it's an error
		return fmt.Errorf("etherscan API error: %s", apiResp.Message)
	}

	// Decode the result into the output struct
	if err := json.Unmarshal(apiResp.Result, out); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// EtherscanTransaction represents a transaction from Etherscan API
type EtherscanTransaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	TransactionIndex  string `json:"transactionIndex"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	IsError           string `json:"isError"`
	TxReceiptStatus   string `json:"txreceipt_status"`
	Input             string `json:"input"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	Confirmations     string `json:"confirmations"`
}

// EtherscanTokenTransfer represents an ERC-20 token transfer from Etherscan API
type EtherscanTokenTransfer struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	From              string `json:"from"`
	ContractAddress   string `json:"contractAddress"`
	To                string `json:"to"`
	Value             string `json:"value"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Input             string `json:"input"`
	Confirmations     string `json:"confirmations"`
}

// EtherscanInternalTransaction represents an internal transaction from Etherscan API
type EtherscanInternalTransaction struct {
	BlockNumber     string `json:"blockNumber"`
	TimeStamp       string `json:"timeStamp"`
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	ContractAddress string `json:"contractAddress"`
	Input           string `json:"input"`
	Type            string `json:"type"`
	Gas             string `json:"gas"`
	GasUsed         string `json:"gasUsed"`
	TraceID         string `json:"traceId"`
	IsError         string `json:"isError"`
	ErrCode         string `json:"errCode"`
}

// GetNormalTransactions fetches normal ETH transactions for an address
func (c *Client) GetNormalTransactions(ctx context.Context, address string, startBlock, endBlock int64, page, offset int, sort string) ([]EtherscanTransaction, error) {
	params := map[string]string{
		"module":  "account",
		"action":  "txlist",
		"address": address,
		"sort":    sort, // "asc" or "desc"
	}

	if startBlock > 0 {
		params["startblock"] = strconv.FormatInt(startBlock, 10)
	}
	if endBlock > 0 {
		params["endblock"] = strconv.FormatInt(endBlock, 10)
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var result []EtherscanTransaction
	if err := c.Get(ctx, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTokenTransfers fetches ERC-20 token transfers for an address
func (c *Client) GetTokenTransfers(ctx context.Context, address string, contractAddress string, startBlock, endBlock int64, page, offset int, sort string) ([]EtherscanTokenTransfer, error) {
	params := map[string]string{
		"module":  "account",
		"action":  "tokentx",
		"address": address,
		"sort":    sort,
	}

	if contractAddress != "" {
		params["contractaddress"] = contractAddress
	}
	if startBlock > 0 {
		params["startblock"] = strconv.FormatInt(startBlock, 10)
	}
	if endBlock > 0 {
		params["endblock"] = strconv.FormatInt(endBlock, 10)
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var result []EtherscanTokenTransfer
	if err := c.Get(ctx, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetInternalTransactions fetches internal transactions for an address
func (c *Client) GetInternalTransactions(ctx context.Context, address string, startBlock, endBlock int64, page, offset int, sort string) ([]EtherscanInternalTransaction, error) {
	params := map[string]string{
		"module":  "account",
		"action":  "txlistinternal",
		"address": address,
		"sort":    sort,
	}

	if startBlock > 0 {
		params["startblock"] = strconv.FormatInt(startBlock, 10)
	}
	if endBlock > 0 {
		params["endblock"] = strconv.FormatInt(endBlock, 10)
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var result []EtherscanInternalTransaction
	if err := c.Get(ctx, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTokenBalance fetches ERC-20 token balance for an address
func (c *Client) GetTokenBalance(ctx context.Context, contractAddress, address string) (string, error) {
	params := map[string]string{
		"module":          "account",
		"action":          "tokenbalance",
		"contractaddress": contractAddress,
		"address":         address,
		"tag":             "latest",
	}

	var result string
	if err := c.Get(ctx, params, &result); err != nil {
		return "", err
	}

	return result, nil
}

// ParseTimestamp converts Etherscan timestamp string to time.Time
func ParseTimestamp(ts string) (time.Time, error) {
	unixTime, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	return time.Unix(unixTime, 0), nil
}
