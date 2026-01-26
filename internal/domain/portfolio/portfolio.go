package portfolio

import (
	"context"
	"errors"
	"testtask/internal/domain/holding"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPortfolioNotFound      = errors.New("portfolio not found")
	ErrPortfolioAddressExists = errors.New("portfolio address already exists")
)

type Portfolio struct {
	ID        string
	Address   string
	Holdings  []*holding.Holding
	UpdatedAt time.Time
}

func NewPortfolio(id, address string) *Portfolio {
	if id == "" {
		id = uuid.New().String()
	}
	return &Portfolio{
		ID:        id,
		Address:   address,
		Holdings:  make([]*holding.Holding, 0),
		UpdatedAt: time.Now(),
	}
}

type Repository interface {
	SingleFetcher
	SingleCreator
	BulkFetcher
}

type SingleFetcher interface {
	GetByAddress(ctx context.Context, address string) (*Portfolio, error)
	GetByID(ctx context.Context, portfolioID string) (*Portfolio, error)
	GetByIDWithHoldings(ctx context.Context, portfolioID string) (*Portfolio, error)
}

type SingleCreator interface {
	Create(ctx context.Context, portfolio *Portfolio) error
}

type BulkFetcher interface {
	List(ctx context.Context) ([]*Portfolio, error)
	ListWithHoldings(ctx context.Context) ([]*Portfolio, error)
}
