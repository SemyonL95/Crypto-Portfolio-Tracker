package price

import (
	"context"
	"math/big"
	"testtask/internal/domain/token"
	"time"
)

const CurrencyDecimal = 8

type Provider interface {
	GetPrices(
		ctx context.Context,
		tokens []*token.Token,
		currency string,
	) (map[*token.Token]*Price, error)
}

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
