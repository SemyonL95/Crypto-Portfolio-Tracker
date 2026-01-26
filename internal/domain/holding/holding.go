package holding

import (
	"context"
	"errors"
	"math/big"
	"testtask/internal/domain/token"
	"time"

	"github.com/google/uuid"
)

var ErrHoldingNotFound = errors.New("holdingRepo.Holding not found")

type Repository interface {
	GetHolding(ctx context.Context, portfolioID string, holdingID string) (*Holding, error)
	CreateHolding(ctx context.Context, portfolioID string, holding *Holding) error
	UpdateHolding(ctx context.Context, portfolioID string, holding *Holding) error
	DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error
	ListByPortfolioID(ctx context.Context, portfolioID string) ([]*Holding, error)
}

type BulkFetcher interface {
}

type Holding struct {
	ID          string
	PortfolioID string
	ChainID     uint8
	Token       *token.Token
	Amount      *big.Int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewHolding(portfolioID string, id string, token *token.Token, amount *big.Int) *Holding {
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
