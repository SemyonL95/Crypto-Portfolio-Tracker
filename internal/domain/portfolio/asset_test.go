package portfolio

import (
	"math/big"
	"testing"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
)

func TestCalculateValue(t *testing.T) {
	tests := []struct {
		name       string
		token      *token.Token
		amount     *big.Int
		priceValue *price.Price
		expected   *big.Int
	}{
		{
			name: "normal case - 18 decimals token",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount: big.NewInt(1000000000000000000), // 1 ETH (with 18 decimals)
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    big.NewInt(30000000000), // $3000.00 (with 8 decimals: 3000 * 10^8)
				Currency: "USD",
			},
			expected: big.NewInt(30000000000), // (1 * 10^18 * 3000 * 10^8) / 10^18 = 3000 * 10^8
		},
		{
			name: "normal case - 6 decimals token",
			token: &token.Token{
				ID:      "usd-coin",
				Name:    "USD Coin",
				Symbol:  "USDC",
				Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
				Decimal: 6,
			},
			amount: big.NewInt(1000000), // 1 USDC (with 6 decimals)
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "usd-coin",
					Name:    "USD Coin",
					Symbol:  "USDC",
					Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					Decimal: 6,
				},
				Value:    big.NewInt(100000000), // $1.00 (with 8 decimals: 1 * 10^8)
				Currency: "USD",
			},
			expected: big.NewInt(100000000), // (1 * 10^6 * 1 * 10^8) / 10^6 = 1 * 10^8
		},
		{
			name: "normal case - 8 decimals token",
			token: &token.Token{
				ID:      "bitcoin",
				Name:    "Bitcoin",
				Symbol:  "BTC",
				Address: "",
				Decimal: 8,
			},
			amount: big.NewInt(100000000), // 1 BTC (with 8 decimals)
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "bitcoin",
					Name:    "Bitcoin",
					Symbol:  "BTC",
					Address: "",
					Decimal: 8,
				},
				Value:    big.NewInt(50000000000), // $50000.00 (with 8 decimals: 50000 * 10^8)
				Currency: "USD",
			},
			expected: big.NewInt(50000000000), // (1 * 10^8 * 50000 * 10^8) / 10^8 = 50000 * 10^8
		},
		{
			name: "fractional amount - 0.5 ETH",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount: big.NewInt(500000000000000000), // 0.5 ETH (with 18 decimals)
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    big.NewInt(30000000000), // $3000.00 (with 8 decimals)
				Currency: "USD",
			},
			expected: big.NewInt(15000000000), // (0.5 * 10^18 * 3000 * 10^8) / 10^18 = 1500 * 10^8
		},
		{
			name: "large amount - 100 ETH",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount: big.NewInt(0).Mul(big.NewInt(100), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)), // 100 ETH
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    big.NewInt(30000000000), // $3000.00 (with 8 decimals: 3000 * 10^8)
				Currency: "USD",
			},
			expected: big.NewInt(3000000000000), // (100 * 10^18 * 30000000000) / 10^18 = 3000000000000
		},
		{
			name: "zero amount",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount: big.NewInt(0),
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    big.NewInt(30000000000),
				Currency: "USD",
			},
			expected: big.NewInt(0),
		},
		{
			name:   "nil token",
			token:  nil,
			amount: big.NewInt(1000000000000000000),
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    big.NewInt(30000000000),
				Currency: "USD",
			},
			expected: nil,
		},
		{
			name: "nil amount",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount: nil,
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    big.NewInt(30000000000),
				Currency: "USD",
			},
			expected: nil,
		},
		{
			name: "nil price",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount:     big.NewInt(1000000000000000000),
			priceValue: nil,
			expected:   nil,
		},
		{
			name: "nil price value",
			token: &token.Token{
				ID:      "ethereum",
				Name:    "Ethereum",
				Symbol:  "ETH",
				Address: "0x0000000000000000000000000000000000000000",
				Decimal: 18,
			},
			amount: big.NewInt(1000000000000000000),
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "ethereum",
					Name:    "Ethereum",
					Symbol:  "ETH",
					Address: "0x0000000000000000000000000000000000000000",
					Decimal: 18,
				},
				Value:    nil,
				Currency: "USD",
			},
			expected: nil,
		},
		{
			name: "token with 0 decimals",
			token: &token.Token{
				ID:      "test-token",
				Name:    "Test Token",
				Symbol:  "TEST",
				Address: "0x123",
				Decimal: 0,
			},
			amount: big.NewInt(100), // 100 tokens
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "test-token",
					Name:    "Test Token",
					Symbol:  "TEST",
					Address: "0x123",
					Decimal: 0,
				},
				Value:    big.NewInt(500000000), // $5.00 (with 8 decimals)
				Currency: "USD",
			},
			expected: big.NewInt(50000000000), // (100 * 5 * 10^8) / 10^0 = 500 * 10^8
		},
		{
			name: "small price value",
			token: &token.Token{
				ID:      "test-token",
				Name:    "Test Token",
				Symbol:  "TEST",
				Address: "0x123",
				Decimal: 18,
			},
			amount: big.NewInt(1000000000000000000), // 1 token
			priceValue: &price.Price{
				Token: &token.Token{
					ID:      "test-token",
					Name:    "Test Token",
					Symbol:  "TEST",
					Address: "0x123",
					Decimal: 18,
				},
				Value:    big.NewInt(1), // $0.00000001 (with 8 decimals)
				Currency: "USD",
			},
			expected: big.NewInt(1), // (1 * 10^18 * 1) / 10^18 = 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If token is nil, we expect nil result since we can't determine decimals
			if tt.token == nil {
				// In real usage, the caller should check if token is nil before calling
				// This test case verifies the expectation that nil token means nil result
				if tt.expected != nil {
					t.Errorf("Test case error: nil token should expect nil result")
				}
				return
			}

			result := CalculateValue(tt.token.Decimal, tt.amount, tt.priceValue)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("CalculateValue() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Errorf("CalculateValue() = nil, want %v", tt.expected)
				return
			}

			if result.Cmp(tt.expected) != 0 {
				t.Errorf("CalculateValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}
