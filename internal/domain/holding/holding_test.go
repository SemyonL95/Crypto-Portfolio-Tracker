package holding

import (
	"math/big"
	"testing"
	domainprice "testtask/internal/domain/price"
	"testtask/internal/domain/token"
)

func TestHolding_CalculateValue_EdgeCases(t *testing.T) {
	t.Run("very large numbers", func(t *testing.T) {
		// Test with very large amounts
		amount := new(big.Int)
		amount.Exp(big.NewInt(10), big.NewInt(30), nil) // 10^30

		holding := &Holding{
			Token: &token.Token{
				Decimal: 18,
				Symbol:  "ETH",
			},
			Amount: amount,
		}

		priceValue := &domainprice.Price{
			Value: big.NewInt(100000000), // $1.00
		}

		result := holding.CalculateValue(priceValue)
		if result == nil {
			t.Error("CalculateValue() returned nil for valid large numbers")
		}

		// Expected: (10^30 * 10^8) / 10^18 = 10^20
		expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
		if result.Cmp(expected) != 0 {
			t.Errorf("CalculateValue() = %v, want %v", result, expected)
		}
	})

	t.Run("zero price", func(t *testing.T) {
		holding := &Holding{
			Token: &token.Token{
				Decimal: 18,
				Symbol:  "ETH",
			},
			Amount: big.NewInt(1000000000000000000), // 1 ETH
		}

		priceValue := &domainprice.Price{
			Value: big.NewInt(0), // $0.00
		}

		result := holding.CalculateValue(priceValue)
		if result == nil {
			t.Error("CalculateValue() returned nil for zero price")
		}

		if result.Sign() != 0 {
			t.Errorf("CalculateValue() = %v, want 0", result)
		}
	})

	t.Run("negative amount should still calculate", func(t *testing.T) {
		// Note: In practice, amounts shouldn't be negative, but we test the function behavior
		holding := &Holding{
			Token: &token.Token{
				Decimal: 18,
				Symbol:  "ETH",
			},
			Amount: big.NewInt(-1000000000000000000), // -1 ETH
		}

		priceValue := &domainprice.Price{
			Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
		}

		result := holding.CalculateValue(priceValue)
		if result == nil {
			t.Error("CalculateValue() returned nil for negative amount")
		}

		// Should return negative value: (-10^18 * 300000000000) / 10^18 = -300000000000
		expected := big.NewInt(-300000000000)
		if result.Cmp(expected) != 0 {
			t.Errorf("CalculateValue() = %v, want %v", result, expected)
		}
	})
}

func TestHolding_CalculateValue_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name        string
		tokenSymbol string
		decimals    uint8
		amount      string // Human-readable amount
		priceUSD    string // Human-readable price
		expectedUSD string // Expected result in USD
	}{
		{
			name:        "Bitcoin example",
			tokenSymbol: "BTC",
			decimals:    8,
			amount:      "0.5",
			priceUSD:    "50000",
			expectedUSD: "25000",
		},
		{
			name:        "Ethereum example",
			tokenSymbol: "ETH",
			decimals:    18,
			amount:      "2.5",
			priceUSD:    "3000",
			expectedUSD: "7500",
		},
		{
			name:        "USDC stablecoin",
			tokenSymbol: "USDC",
			decimals:    6,
			amount:      "1000",
			priceUSD:    "1",
			expectedUSD: "1000",
		},
		{
			name:        "Small altcoin",
			tokenSymbol: "ALT",
			decimals:    18,
			amount:      "1000",
			priceUSD:    "0.001",
			expectedUSD: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert human-readable amount to big.Int
			amountFloat := new(big.Float)
			amountFloat.SetString(tt.amount)
			decimalsMultiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tt.decimals)), nil))
			amountFloat.Mul(amountFloat, decimalsMultiplier)
			amount, _ := amountFloat.Int(nil)

			// Convert human-readable price to big.Int (with 8 decimals)
			priceFloat := new(big.Float)
			priceFloat.SetString(tt.priceUSD)
			priceMultiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil))
			priceFloat.Mul(priceFloat, priceMultiplier)
			price, _ := priceFloat.Int(nil)

			// Convert expected USD to big.Int (with 8 decimals)
			expectedFloat := new(big.Float)
			expectedFloat.SetString(tt.expectedUSD)
			expectedFloat.Mul(expectedFloat, priceMultiplier)
			expected, _ := expectedFloat.Int(nil)

			holding := &Holding{
				Token: &token.Token{
					Decimal: tt.decimals,
					Symbol:  tt.tokenSymbol,
				},
				Amount: amount,
			}

			priceValue := &domainprice.Price{
				Value:    price,
				Currency: "USD",
				Token:    holding.Token,
			}

			result := holding.CalculateValue(priceValue)

			if result == nil {
				t.Fatalf("CalculateValue() returned nil")
			}

			// Allow small rounding differences (within 1 cent)
			diff := new(big.Int).Sub(result, expected)
			diff.Abs(diff)
			maxDiff := big.NewInt(1000000) // 0.01 USD in smallest units

			if diff.Cmp(maxDiff) > 0 {
				t.Errorf("CalculateValue() = %v, want approximately %v (diff: %v)", result, expected, diff)
			}
		})
	}
}
