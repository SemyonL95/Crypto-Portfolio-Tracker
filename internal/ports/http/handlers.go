package http

import (
	"context"
	"math/big"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
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

type TransactionService interface {
	GetTransactions(ctx context.Context, filters TransactionFilters) ([]transaction.Transaction, int, error)
	GetTransactionByHash(ctx context.Context, hash string) (*transaction.Transaction, error)
}

type PriceService interface {
	GetPrices(ctx context.Context, tokens []*price.Token, currency string) (map[*price.Token]*price.Price, error)
}

type PortfolioService interface {
	CreatePortfolio(ctx context.Context, portfolio *portfolio.Portfolio) error
	GetPortfolio(ctx context.Context, portfolioID string) (*portfolio.Portfolio, error)
	AddHolding(ctx context.Context, userID string, holding *portfolio.Holding) error
	UpdateHolding(ctx context.Context, userID string, holdingID string, amount *big.Int) error
	DeleteHolding(ctx context.Context, userID string, holdingID string) error
}

type TokensService interface {
	GetTokenByAddress(_ context.Context, address string) (*price.Token, bool)
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
	Method       string    `json:"method"`
	Timestamp    time.Time `json:"timestamp"`
	BlockNumber  int64     `json:"block_number"`
}

type Holding struct {
	ID           string    `json:"id"`
	TokenAddress string    `json:"token_address"`
	TokenSymbol  string    `json:"token_symbol"`
	Amount       *big.Int  `json:"amount"`
	ValueUSD     *big.Int  `json:"value_usd"`
	PriceUSD     *big.Int  `json:"price_usd"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Portfolio struct {
	ID        string     `json:"id"`
	Holdings  []*Holding `json:"holdings"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Price struct {
	TokenID     string  `json:"token_id"`
	Symbol      string  `json:"symbol"`
	PriceUSD    float64 `json:"price_usd"`
	LastUpdated int64   `json:"last_updated"`
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
