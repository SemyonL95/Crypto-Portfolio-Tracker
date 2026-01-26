package portfolio

import (
	"math/big"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
	"time"
)

// Asset represents an asset from holdings or transactions with its value
type Asset struct {
	Token  *token.Token
	Amount *big.Int
	Price  *price.Price
	Value  *big.Int
	Source string // "asset" or "transaction"
}

// PortfolioAssets represents all assets in a portfolio with their values
type PortfolioAssets struct {
	PortfolioID  string
	Address      string
	Assets       []*Asset
	TotalValue   *big.Int
	Currency     string
	CalculatedAt time.Time
}

// CalculateValue calculates the USD value of a asset based on token price, decimals, and amount.
// The calculation accounts for token decimals but keeps the result in smallest currency units (8 decimals for USD).
// Formula: (amount * price) / 10^tokenDecimals
// Returns nil if any required field is missing or nil.
// The result is in smallest currency units (e.g., cents for USD with 8 decimals).
func CalculateValue(amount *big.Int, t *token.Token, priceValue *price.Price) *big.Int {
	if amount == nil || t == nil || priceValue == nil || priceValue.Value == nil {
		return nil
	}

	// If amount is zero, return zero
	if amount.Sign() == 0 {
		return big.NewInt(0)
	}

	// Calculate: (amount * price) / 10^tokenDecimals
	// This removes token decimals but keeps currency decimals (8 for USD)
	result := new(big.Int).Mul(amount, priceValue.Value)

	// Create divisor: 10^tokenDecimals
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(t.Decimal)), nil)

	// Divide to get the final value (still in smallest currency units)
	result.Div(result, divisor)

	return result
}
