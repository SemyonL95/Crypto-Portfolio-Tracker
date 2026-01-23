package price

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testtask/internal/domain/price"
	"time"
)

// MockProvider implements PriceProvider with mock data for testing/fallback
type MockProvider struct {
}

// NewMockProvider creates a new mock price provider
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// GetPrices retrieves prices for multiple tokens (batch request)
// Implements PriceProvider interface from domain
func (p *MockProvider) GetPrices(
	ctx context.Context,
	tokens []*price.Token,
	currency string,
) (map[*price.Token]*price.Price, error) {
	results := make(map[*price.Token]*price.Price)

	for _, token := range tokens {
		results[token] = &price.Price{
			Token:       token,
			Value:       big.NewInt(1000000000),
			Currency:    "",
			LastUpdated: time.Now(),
		}
	}

	return results, nil
}

// addPriceVariation adds small random variation to price for realism
func (p *MockProvider) addPriceVariation(basePrice float64) float64 {
	// Add ±1% variation
	variation := (rand.Float64() - 0.5) * 0.02 // -1% to +1%
	return basePrice * (1 + variation)
}

// convertFloatToBigInt converts a float64 price to big.Int with specified decimals
// This is a package-level utility function
func convertFloatToBigInt(value float64, decimals int) *big.Int {
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	// Convert float to string with enough precision, then to big.Float
	valueStr := fmt.Sprintf("%.8f", value)
	valueFloat, _, _ := big.ParseFloat(valueStr, 10, 256, big.ToNearestEven)
	multiplierFloat := new(big.Float).SetInt(multiplier)
	resultFloat := new(big.Float).Mul(valueFloat, multiplierFloat)
	result, _ := resultFloat.Int(nil)
	return result
}
