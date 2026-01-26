package http

import (
	"context"
	"math/big"
	portfolioService "testtask/internal/application/portfolio"
	"testtask/internal/domain/holding"
	domainPortfolio "testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
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
	// GetTransactions returns paginated transactions for an address
	GetTransactions(ctx context.Context, filters TransactionFilters) ([]transaction.Transaction, int, error)
	// GetAllTransactions returns all transactions for an address without pagination
	GetAllTransactions(ctx context.Context, filters TransactionFilters) ([]transaction.Transaction, error)
	GetTransactionByHash(ctx context.Context, hash string) (*transaction.Transaction, error)
}

type PriceService interface {
	GetPrices(ctx context.Context, tokens []*token.Token, currency string) (map[*token.Token]*price.Price, error)
}

type PortfolioService interface {
	CreatePortfolio(ctx context.Context, portfolio *domainPortfolio.Portfolio) error
	GetPortfolio(ctx context.Context, portfolioID string) (*domainPortfolio.Portfolio, error)
	AddHolding(ctx context.Context, userID string, holding *holding.Holding) error
	UpdateHolding(ctx context.Context, userID string, holdingID string, amount *big.Int) error
	DeleteHolding(ctx context.Context, userID string, holdingID string) error
	GetPortfolioAssets(ctx context.Context, portfolioID string, currency string) (*portfolioService.PortfolioAssets, error)
}

type TokensService interface {
	GetTokenByAddress(_ context.Context, address string) (*token.Token, bool)
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
	Holdings  []*Holding `json:"holdingRepo"`
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

// Asset represents an asset in the portfolio with its value
type Asset struct {
	Token       *TokenInfo `json:"token"`
	Amount      *big.Int   `json:"amount"`
	PriceUSD    *big.Int   `json:"price_usd"`
	ValueUSD    *big.Int   `json:"value_usd"`
	Source      string     `json:"source"` // "holding" or "transaction"
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
	PortfolioID  string    `json:"portfolio_id"`
	Address      string    `json:"address"`
	Assets       []*Asset  `json:"assets"`
	TotalValue   *big.Int  `json:"total_value"`
	Currency     string    `json:"currency"`
	CalculatedAt time.Time `json:"calculated_at"`
}
