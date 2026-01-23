package portfolio

import (
	"context"
	"errors"
	"math/big"
	"testtask/internal/domain/price"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPortfolioNotFound      = errors.New("portfolio not found")
	ErrHoldingNotFound        = errors.New("holding not found")
	ErrPortfolioAddressExists = errors.New("portfolio address already exists")
)

// Holding represents a token holding in the portfolio
type Holding struct {
	ID          string
	PortfolioID string
	Token       *price.Token
	Amount      *big.Int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Portfolio represents the user's crypto portfolio
type Portfolio struct {
	ID        string
	Address   string
	Holdings  []*Holding
	UpdatedAt time.Time
}

func NewPortfolio(id, address string) *Portfolio {
	if id == "" {
		id = uuid.New().String()
	}
	return &Portfolio{
		ID:        id,
		Address:   address,
		Holdings:  make([]*Holding, 0),
		UpdatedAt: time.Now(),
	}
}

func NewHolding(portfolioID string, id string, token *price.Token, amount *big.Int) *Holding {
	if id == "" {
		id = uuid.New().String()
	}
	now := time.Now()
	return &Holding{
		ID:          id,
		PortfolioID: portfolioID,
		Token:       token,
		Amount:      new(big.Int).Set(amount),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

type PortfolioRepository interface {
	Get(ctx context.Context, portfolioID string) (*Portfolio, error)
	GetByAddress(ctx context.Context, address string) (*Portfolio, error)
	Save(ctx context.Context, portfolio *Portfolio) error
	GetHolding(ctx context.Context, portfolioID string, holdingID string) (*Holding, error)
	AddHolding(ctx context.Context, portfolioID string, holding *Holding) error
	UpdateHolding(ctx context.Context, portfolioID string, holding *Holding) error
	DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error
}
