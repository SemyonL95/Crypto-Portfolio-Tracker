package price

import (
	"context"
	"math/big"
	"testtask/internal/domain/token"
	"time"
)

const CurrencyDecimal = 8

type Cache[K string, P Price] interface {
	GetBatch(ctx context.Context, tokenIDs []string) (map[K]*P, bool)
	SetBatch(ctx context.Context, prices map[K]*P) bool
}

type PriceProvider interface {
	GetPrices(
		ctx context.Context,
		tokens []*token.Token,
		currency string,
	) (map[*token.Token]*Price, error)
}

// Provider is an alias for PriceProvider for backward compatibility.
type Provider = PriceProvider

type Price struct {
	Token       *token.Token
	Value       *big.Int
	Currency    string
	LastUpdated time.Time
}

func NewPrice(token *token.Token, amount *big.Int, currency string) Price {
	return Price{
		Token:       token,
		Value:       amount,
		Currency:    currency,
		LastUpdated: time.Now(),
	}
}
