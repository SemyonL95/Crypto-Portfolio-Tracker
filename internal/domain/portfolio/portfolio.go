package portfolio

import (
	"context"
	"errors"
	domainHolding "testtask/internal/domain/holding"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPortfolioNotFound      = errors.New("portfolio not found")
	ErrPortfolioAddressExists = errors.New("portfolio address already exists")
)

// holdingRepo.Holding represents a token holdingRepo.Holding in the portfolio

// Portfolio represents the user's crypto portfolio
type Portfolio struct {
	ID        string
	Address   string
	Holdings  []*domainHolding.Holding
	UpdatedAt time.Time
}

func NewPortfolio(id, address string) *Portfolio {
	if id == "" {
		id = uuid.New().String()
	}
	return &Portfolio{
		ID:        id,
		Address:   address,
		Holdings:  make([]*domainHolding.Holding, 0),
		UpdatedAt: time.Now(),
	}
}

type Repository interface {
	GetByAddress(ctx context.Context, address string) (*Portfolio, error)
	GetByID(ctx context.Context, portfolioID string) (*Portfolio, error)
	GetByIDWithHoldings(ctx context.Context, portfolioID string) (*Portfolio, error)
	Create(ctx context.Context, portfolio *Portfolio) error
	List(ctx context.Context) ([]*Portfolio, error)
}
