package coingecko

import (
	"context"
	"math/big"
	"math/rand"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
	"time"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) GetPrices(
	ctx context.Context,
	tokens []*token.Token,
	currency string,
) (map[*token.Token]*price.Price, error) {
	results := make(map[*token.Token]*price.Price)

	for _, token := range tokens {
		results[token] = &price.Price{
			Token:       token,
			Value:       big.NewInt(1000000000),
			Currency:    currency,
			LastUpdated: time.Now(),
		}
	}

	return results, nil
}

func (p *MockProvider) addPriceVariation(basePrice float64) float64 {
	// Add ±1% variation
	variation := (rand.Float64() - 0.5) * 0.02 // -1% to +1%
	return basePrice * (1 + variation)
}

func convertFloatToBigInt(value float64, decimals int) *big.Int {
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	// Convert float to big.Float directly to preserve precision
	valueFloat := new(big.Float).SetFloat64(value)
	multiplierFloat := new(big.Float).SetInt(multiplier)
	resultFloat := new(big.Float).Mul(valueFloat, multiplierFloat)
	result, _ := resultFloat.Int(nil)
	return result
}
