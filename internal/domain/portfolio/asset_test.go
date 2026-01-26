package portfolio

import (
	"math/big"
	"testing"
	domainprice "testtask/internal/domain/price"
	"testtask/internal/domain/token"
)

func TestAsset_CalculateValue(t *testing.T) {
	tests := []struct {
		name        string
		asset       *Asset
		priceValue  *domainprice.Price
		expected    *big.Int
		expectNil   bool
		description string
	}{
		{
			name: "standard calculation - 18 decimals token",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(1000000000000000000), // 1 ETH (1 * 10^18)
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			expectNil:   false,
			description: "1 ETH at $3000 should equal $3000",
		},
		{
			name: "standard calculation - 6 decimals token",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 6,
					Symbol:  "USDC",
				},
				Amount: big.NewInt(1000000), // 1 USDC (1 * 10^6)
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(100000000), // $1.00 (1 * 10^8)
			},
			expected:    big.NewInt(100000000), // $1.00 (1 * 10^8)
			expectNil:   false,
			description: "1 USDC at $1.00 should equal $1.00",
		},
		{
			name: "fractional amount - 0.5 ETH",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(500000000000000000), // 0.5 ETH (0.5 * 10^18)
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    big.NewInt(150000000000), // $1500.00 (1500 * 10^8)
			expectNil:   false,
			description: "0.5 ETH at $3000 should equal $1500",
		},
		{
			name: "large amount - 100 ETH",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(0).Mul(big.NewInt(100), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)), // 100 ETH
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    big.NewInt(30000000000000), // $300,000.00 (300000 * 10^8)
			expectNil:   false,
			description: "100 ETH at $3000 should equal $300,000",
		},
		{
			name: "small price - $0.01",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "TOKEN",
				},
				Amount: big.NewInt(1000000000000000000), // 1 token
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(1000000), // $0.01 (0.01 * 10^8)
			},
			expected:    big.NewInt(1000000), // $0.01 (0.01 * 10^8)
			expectNil:   false,
			description: "1 token at $0.01 should equal $0.01",
		},
		{
			name: "zero amount",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(0),
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    big.NewInt(0),
			expectNil:   false,
			description: "Zero amount should return zero value",
		},
		{
			name:  "nil asset",
			asset: nil,
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    nil,
			expectNil:   true,
			description: "Nil asset should return nil",
		},
		{
			name: "nil amount",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: nil,
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    nil,
			expectNil:   true,
			description: "Nil amount should return nil",
		},
		{
			name: "nil token",
			asset: &Asset{
				Token:  nil,
				Amount: big.NewInt(1000000000000000000),
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    nil,
			expectNil:   true,
			description: "Nil token should return nil",
		},
		{
			name: "nil price",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(1000000000000000000),
			},
			priceValue:  nil,
			expected:    nil,
			expectNil:   true,
			description: "Nil price should return nil",
		},
		{
			name: "nil price value",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(1000000000000000000),
			},
			priceValue: &domainprice.Price{
				Value: nil,
			},
			expected:    nil,
			expectNil:   true,
			description: "Nil price value should return nil",
		},
		{
			name: "8 decimals token",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 8,
					Symbol:  "TOKEN",
				},
				Amount: big.NewInt(100000000), // 1 token (1 * 10^8)
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(500000000), // $5.00 (5 * 10^8)
			},
			expected:    big.NewInt(500000000), // $5.00 (5 * 10^8)
			expectNil:   false,
			description: "1 token (8 decimals) at $5.00 should equal $5.00",
		},
		{
			name: "very small amount",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(1000000000), // 0.000000001 ETH (1 * 10^9 wei)
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(300000000000), // $3000.00 (3000 * 10^8)
			},
			expected:    big.NewInt(300), // $0.000003 (0.000000001 ETH * $3000 / 10^18)
			expectNil:   false,
			description: "Very small amount should calculate correctly",
		},
		{
			name: "high precision calculation",
			asset: &Asset{
				Token: &token.Token{
					Decimal: 18,
					Symbol:  "ETH",
				},
				Amount: big.NewInt(123456789000000000), // 0.123456789 ETH
			},
			priceValue: &domainprice.Price{
				Value: big.NewInt(250000000000), // $2500.00 (2500 * 10^8)
			},
			expected:    big.NewInt(30864197250), // $308.6419725 (0.123456789 ETH * $2500 / 10^18)
			expectNil:   false,
			description: "High precision calculation should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *big.Int
			if tt.asset == nil {
				result = CalculateValue(nil, nil, tt.priceValue)
			} else {
				result = CalculateValue(tt.asset.Amount, tt.asset.Token, tt.priceValue)
			}

			if tt.expectNil {
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
				t.Errorf("CalculateValue() = %v, want %v (%s)", result, tt.expected, tt.description)
			}
		})
	}
}
