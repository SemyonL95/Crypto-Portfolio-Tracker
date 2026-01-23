package price

import (
	"context"
	"math/big"
	"time"
)

const CurrencyDecimal = 8

type Token struct {
	ID      string
	Symbol  string
	Address string
}

type PriceCache interface {
	GetPrice(ctx context.Context, tokenID string, currency string) (*Price, bool)
	SetPrice(ctx context.Context, tokenID string, price *Price, currency string) bool
	GetPrices(ctx context.Context, tokenIDs []string, currency string) (map[string]*Price, error)
	SetPrices(ctx context.Context, prices map[*Token]*Price, currency string) error
}

type PriceProvider interface {
	GetPrices(
		ctx context.Context,
		tokens []*Token,
		currency string,
	) (map[*Token]*Price, error)
}

type TokenProvider interface {
	GetTokenByAddress(address string) (*Token, bool)
}

type Price struct {
	Token       *Token
	Value       *big.Int
	Currency    string
	LastUpdated time.Time
}

func NewPrice(token *Token, amount *big.Int, currency string) Price {
	return Price{
		Token:       token,
		Value:       amount,
		Currency:    currency,
		LastUpdated: time.Now(),
	}
}

func (p *Price) CalculatePriceForTokensAmount(tokenAmounts *big.Int) *big.Int {
	if p.Value == nil {
		return big.NewInt(0)
	}

	result := new(big.Int).Mul(p.Value, tokenAmounts)
	return result
}
