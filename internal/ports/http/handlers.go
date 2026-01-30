package http

import (
	"fmt"
	"math/big"
	"testtask/internal/domain/transaction"
	"time"
)

type TransactionFilters struct {
	Address  *string    `json:"address"`
	Type     *string    `json:"type"`
	Status   *string    `json:"status"`
	Token    *string    `json:"token"`
	FromDate *time.Time `json:"from_date"`
	ToDate   *time.Time `json:"to_date"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type Transaction struct {
	ID           string    `json:"id"`
	Hash         string    `json:"hash"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	TokenAddress string    `json:"token_address"`
	TokenSymbol  string    `json:"token_symbol"`
	Amount       string    `json:"amount"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Direction    string    `json:"direction"`
	Method       string    `json:"method"`
	Timestamp    time.Time `json:"timestamp"`
	BlockNumber  int64     `json:"block_number"`
}

type Holding struct {
	ID           string   `json:"id"`
	TokenAddress string   `json:"token_address"`
	TokenSymbol  string   `json:"token_symbol"`
	Amount       *big.Int `json:"amount"`
}

// ToDomainFilterOptions maps HTTP transaction filters to domain filter options.
func ToDomainFilterOptions(f TransactionFilters) (transaction.FilterOptions, error) {
	var opts transaction.FilterOptions

	if f.Address != nil {
		opts.Address = *f.Address
	}

	if f.Type != nil && *f.Type != "" {
		t := transaction.TransactionType(*f.Type)
		switch t {
		case transaction.TransactionTypeSend,
			transaction.TransactionTypeReceive,
			transaction.TransactionTypeSwap,
			transaction.TransactionTypeStake:
			opts.Type = &t
		default:
			return opts, fmt.Errorf("invalid transaction type: %s", *f.Type)
		}
	}

	if f.Status != nil && *f.Status != "" {
		s := transaction.TransactionStatus(*f.Status)
		switch s {
		case transaction.TransactionStatusPending,
			transaction.TransactionStatusSuccess,
			transaction.TransactionStatusFailed:
			opts.Status = &s
		default:
			return opts, fmt.Errorf("invalid transaction status: %s", *f.Status)
		}
	}

	if f.Token != nil && *f.Token != "" {
		opts.Token = f.Token
	}
	if f.FromDate != nil {
		opts.FromDate = f.FromDate
	}
	if f.ToDate != nil {
		opts.ToDate = f.ToDate
	}

	opts.Page = f.Page
	opts.PageSize = f.PageSize

	return opts, nil
}

type Portfolio struct {
	ID       string     `json:"id"`
	Holdings []*Holding `json:"holdingRepo"`
}

type Price struct {
	TokenID  string  `json:"token_id"`
	Symbol   string  `json:"symbol"`
	PriceUSD float64 `json:"price_usd"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// Asset represents an asset in the portfolio with its value
type Asset struct {
	Token    *TokenInfo `json:"token"`
	Amount   *big.Int   `json:"amount"`
	PriceUSD float64    `json:"price_usd"`
	ValueUSD float64    `json:"value_usd"`
	Source   string     `json:"source"` // "holding" or "transaction"
}

// TokenInfo represents token information in the response
type TokenInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Address string `json:"address"`
	Decimal uint8  `json:"decimal"`
}

// PortfolioAssets represents all assets in a portfolio with their values
type PortfolioAssets struct {
	PortfolioID string   `json:"portfolio_id"`
	Address     string   `json:"address"`
	Assets      []*Asset `json:"assets"`
}
