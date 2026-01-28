package etherscan

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"

	"testtask/internal/application/ratelimiter"
	"testtask/internal/domain"
	"testtask/internal/domain/transaction"
)

// generic Etherscan API envelope.
type apiResponse[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  T      `json:"result"`
}

// Normal transaction response.
type normalTx struct {
	BlockNumber     string `json:"blockNumber"`
	TimeStamp       string `json:"timeStamp"`
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	GasPrice        string `json:"gasPrice"`
	GasUsed         string `json:"gasUsed"`
	Input           string `json:"input"`
	MethodID        string `json:"methodId"`     // 0x+4byte selector (optional)
	FunctionName    string `json:"functionName"` // human readable (optional)
	IsError         string `json:"isError"`      // "0" success, "1" error
	TxReceiptStatus string `json:"txreceipt_status"`
}

// Internal transaction response.
type internalTx struct {
	BlockNumber string `json:"blockNumber"`
	TimeStamp   string `json:"timeStamp"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	IsError     string `json:"isError"`
}

// ERC-20 token transfer response.
type tokenTx struct {
	BlockNumber     string `json:"blockNumber"`
	TimeStamp       string `json:"timeStamp"`
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	ContractAddress string `json:"contractAddress"`
	TokenSymbol     string `json:"tokenSymbol"`
	TokenDecimal    string `json:"tokenDecimal"`
	Value           string `json:"value"`
}

// Provider implements transaction.Provider and transaction.Repository
// using an Etherscan-compatible API.
// It keeps all pagination and HTTP concerns internal.
type Provider struct {
	client      *Client
	rateLimiter domain.RateLimiterService
}

func NewProvider(client *Client, rl *ratelimiter.RateLimiter) *Provider {
	return &Provider{
		client:      client,
		rateLimiter: rl,
	}
}

const (
	defaultPageSize = 1000
	maxPageSize     = 10000
)

func (p *Provider) NativeTxsByAddress(ctx context.Context, address string, opts transaction.FilterOptions) ([]*transaction.Transaction, error) {
	if err := p.allow(ctx); err != nil {
		return nil, err
	}

	page, pageSize := normalizePage(opts.Page, opts.PageSize)
	addr := strings.ToLower(strings.TrimSpace(address))

	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "txlist")
	params.Set("address", addr)
	params.Set("sort", "asc")
	params.Set("page", strconv.Itoa(page))
	params.Set("offset", strconv.Itoa(pageSize))

	var resp apiResponse[[]normalTx]
	if err := p.client.get(ctx, params, &resp); err != nil {
		return nil, fmt.Errorf("etherscan native txs: %w", err)
	}

	if resp.Status != "1" && resp.Message != "No transactions found" {
		return nil, fmt.Errorf("etherscan native txs: status=%s message=%s", resp.Status, resp.Message)
	}

	return mapNormalTxs(resp.Result, addr), nil
}

func (p *Provider) TokenTxsByAddress(ctx context.Context, address string, opts transaction.FilterOptions) ([]*transaction.Transaction, error) {
	if err := p.allow(ctx); err != nil {
		return nil, err
	}

	page, pageSize := normalizePage(opts.Page, opts.PageSize)
	addr := strings.ToLower(strings.TrimSpace(address))

	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "tokentx")
	params.Set("address", addr)
	params.Set("sort", "asc")
	params.Set("page", strconv.Itoa(page))
	params.Set("offset", strconv.Itoa(pageSize))

	var resp apiResponse[[]tokenTx]
	if err := p.client.get(ctx, params, &resp); err != nil {
		return nil, fmt.Errorf("etherscan token txs: %w", err)
	}
	if resp.Status != "1" && resp.Message != "No transactions found" {
		return nil, fmt.Errorf("etherscan token txs: status=%s message=%s", resp.Status, resp.Message)
	}

	return mapTokenTxs(resp.Result, addr), nil
}

func (p *Provider) InternalTxsByAddress(ctx context.Context, address string, opts transaction.FilterOptions) ([]*transaction.Transaction, error) {
	if err := p.allow(ctx); err != nil {
		return nil, err
	}

	page, pageSize := normalizePage(opts.Page, opts.PageSize)
	addr := strings.ToLower(strings.TrimSpace(address))

	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "txlistinternal")
	params.Set("address", addr)
	params.Set("sort", "asc")
	params.Set("page", strconv.Itoa(page))
	params.Set("offset", strconv.Itoa(pageSize))

	var resp apiResponse[[]internalTx]
	if err := p.client.get(ctx, params, &resp); err != nil {
		return nil, fmt.Errorf("etherscan internal txs: %w", err)
	}
	if resp.Status != "1" && resp.Message != "No transactions found" {
		return nil, fmt.Errorf("etherscan internal txs: status=%s message=%s", resp.Status, resp.Message)
	}

	return mapInternalTxs(resp.Result, addr), nil
}

func (p *Provider) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	if err := p.allow(ctx); err != nil {
		return nil, err
	}

	addr := strings.ToLower(strings.TrimSpace(address))

	params := url.Values{}
	params.Set("module", "proxy")
	params.Set("action", "eth_getBalance")
	params.Set("address", addr)
	params.Set("tag", "latest")

	var resp apiResponse[string]
	if err := p.client.get(ctx, params, &resp); err != nil {
		return nil, fmt.Errorf("etherscan eth_getBalance: %w", err)
	}

	// The result is a hex string like "0x1234..."
	balance := parseHexBig(resp.Result)
	return balance, nil
}

func (p *Provider) allow(ctx context.Context) error {
	if p.rateLimiter == nil {
		return nil
	}
	return p.rateLimiter.Allow(ctx)
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func mapNormalTxs(items []normalTx, address string) []*transaction.Transaction {
	var out []*transaction.Transaction
	for _, it := range items {
		ts := parseUnix(it.TimeStamp)
		gasPrice := parseBig(it.GasPrice)
		gasUsed := parseBig(it.GasUsed)
		amount := parseBig(it.Value)
		blockNum := parseBlockNumber(it.BlockNumber)

		status := transaction.TransactionStatusSuccess
		if it.IsError == "1" || it.TxReceiptStatus == "0" {
			status = transaction.TransactionStatusFailed
		}

		t := &transaction.Transaction{
			ID:          it.Hash,
			Hash:        it.Hash,
			From:        strings.ToLower(it.From),
			To:          strings.ToLower(it.To),
			Amount:      amount,
			Status:      status,
			Method:      it.FunctionName,
			MethodSig:   it.MethodID,
			GasPrice:    gasPrice,
			GasUsed:     gasUsed,
			Timestamp:   ts,
			BlockNumber: blockNum,
		}

		// Set direction based on address
		t.SetDirectionForAddress(address)

		out = append(out, t)
	}
	return out
}

func mapInternalTxs(items []internalTx, address string) []*transaction.Transaction {
	var out []*transaction.Transaction
	for _, it := range items {
		ts := parseUnix(it.TimeStamp)
		amount := parseBig(it.Value)
		blockNum := parseBlockNumber(it.BlockNumber)

		status := transaction.TransactionStatusSuccess
		if it.IsError == "1" {
			status = transaction.TransactionStatusFailed
		}

		t := &transaction.Transaction{
			ID:          it.Hash,
			Hash:        it.Hash,
			From:        strings.ToLower(it.From),
			To:          strings.ToLower(it.To),
			Amount:      amount,
			Status:      status,
			Timestamp:   ts,
			BlockNumber: blockNum,
		}

		// Set direction based on address
		t.SetDirectionForAddress(address)

		out = append(out, t)
	}
	return out
}

func mapTokenTxs(items []tokenTx, address string) []*transaction.Transaction {
	var out []*transaction.Transaction
	for _, it := range items {
		ts := parseUnix(it.TimeStamp)
		amount := parseBig(it.Value)
		blockNum := parseBlockNumber(it.BlockNumber)

		id := fmt.Sprintf("%s:%s", it.Hash, it.ContractAddress)

		t := &transaction.Transaction{
			ID:           id,
			Hash:         it.Hash,
			From:         strings.ToLower(it.From),
			To:           strings.ToLower(it.To),
			TokenAddress: strings.ToLower(it.ContractAddress),
			TokenSymbol:  it.TokenSymbol,
			Amount:       amount,
			Status:       transaction.TransactionStatusSuccess, // Etherscan token transfers are only for successful txs
			Timestamp:    ts,
			BlockNumber:  blockNum,
		}

		// Set direction based on address
		t.SetDirectionForAddress(address)

		out = append(out, t)
	}
	return out
}

func parseUnix(s string) time.Time {
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}

func parseBig(s string) *big.Int {
	if s == "" {
		return big.NewInt(0)
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return v
}

func parseBlockNumber(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// parseHexBig parses a hex string (with or without 0x prefix) to big.Int.
func parseHexBig(s string) *big.Int {
	if s == "" {
		return big.NewInt(0)
	}
	// Remove 0x prefix if present
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")

	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return big.NewInt(0)
	}
	return v
}
