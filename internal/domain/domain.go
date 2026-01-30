package domain

import (
	"context"
	"math/big"
	domainHolding "testtask/internal/domain/holding"
	domainPortfolio "testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
	"testtask/internal/domain/transaction"
)

type RateLimiterService interface {
	Allow(ctx context.Context) error
}

type Cache[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, bool)
	Set(ctx context.Context, key K, value V)
	GetBatch(ctx context.Context, keys []K) map[K]V
	SetBatch(ctx context.Context, items map[K]V)
}

// TransactionService defines the interface for transaction operations.
type TransactionService interface {
	GetTransactions(ctx context.Context, address string, opts transaction.FilterOptions) ([]transaction.Transaction, int, error)
}

type PriceService interface {
	GetPrices(ctx context.Context, tokens []*token.Token, currency string) (map[*token.Token]*price.Price, error)
}

type PortfolioService interface {
	ListPortfolios(ctx context.Context) ([]*domainPortfolio.Portfolio, error)
	CreatePortfolio(ctx context.Context, portfolio *domainPortfolio.Portfolio) error
	GetPortfolio(ctx context.Context, portfolioID string) (*domainPortfolio.Portfolio, error)
	AddHolding(ctx context.Context, userID string, holding *domainHolding.Holding) error
	UpdateHolding(ctx context.Context, userID string, holdingID string, amount *big.Int) error
	DeleteHolding(ctx context.Context, userID string, holdingID string) error
	GetPortfolioAssets(ctx context.Context, portfolioID string, currency string) (*domainPortfolio.Portfolio, []*domainPortfolio.Asset, error)
}

type TokensService interface {
	GetTokenByAddress(_ context.Context, address string) (*token.Token, bool)
}
