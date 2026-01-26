package coingecko

import (
	"context"
	"fmt"
	"strings"
	"testtask/internal/domain/token"
	"time"

	"testtask/internal/domain/price"
)

type CoinGeckoSimplePriceResponse map[string]map[string]float64

type PriceRepository struct {
	coingeckoClient *Client
	symbolToID      map[string]string // Cache for symbol to CoinGecko ID mapping
}

func NewPriceRepository(coingeckoClient *Client, symbolToID map[string]string) *PriceRepository {
	return &PriceRepository{coingeckoClient: coingeckoClient, symbolToID: symbolToID}
}

func (a *PriceRepository) GetPrices(
	ctx context.Context,
	tokens []*token.Token,
	currency string,
) (map[*token.Token]*price.Price, error) {
	if len(tokens) == 0 {
		return make(map[*token.Token]*price.Price), nil
	}

	currency = strings.ToLower(currency)
	if currency == "" {
		currency = "usd"
	}

	tokenMap := make(map[string]*token.Token) // Map from CoinGecko ID to Token
	for _, token := range tokens {
		tokenMap[token.ID] = token
	}

	const maxBatchSize = 250
	results := make(map[*token.Token]*price.Price)

	for i := 0; i < len(tokens); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(tokens) {
			end = len(tokens)
		}

		batchTokens := tokens[i:end]
		var batch []string

		for _, token := range batchTokens {
			batch = append(batch, token.ID)
		}

		batchResults, err := a.fetchPricesBatch(ctx, batch, currency, tokenMap)
		if err != nil {
			return nil, err
		}

		for token, price := range batchResults {
			results[token] = price
		}
	}

	return results, nil
}

// fetchPricesBatch fetches prices for a batch of token IDs
func (a *PriceRepository) fetchPricesBatch(
	ctx context.Context,
	tokenIDs []string,
	currency string,
	tokenMap map[string]*token.Token,
) (map[*token.Token]*price.Price, error) {
	idsParam := strings.Join(tokenIDs, ",")
	path := fmt.Sprintf("/simple/token_price?ids=%s&vs_currencies=%s&include_last_updated_at=true&include_tokens=all&precision=8", idsParam, currency)

	var data CoinGeckoSimplePriceResponse

	err := a.coingeckoClient.Get(ctx, path, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}

	results := make(map[*token.Token]*price.Price)
	for tokenID, priceData := range data {
		if priceValue, ok := priceData[currency]; ok {
			t, ok := tokenMap[tokenID]
			if !ok {
				symbol := a.getSymbolFromID(tokenID)
				t = &token.Token{
					ID:     tokenID,
					Symbol: symbol,
					//Address:
				}
			}

			priceAmount := convertFloatToBigInt(priceValue, price.CurrencyDecimal)

			lastUpdated := time.Now()
			if timestamp, ok := priceData["last_updated_at"]; ok {
				lastUpdated = time.Unix(int64(timestamp), 0)
			}

			p := price.NewPrice(t, priceAmount, strings.ToUpper(currency))

			p.LastUpdated = lastUpdated
			results[t] = &p
		}
	}

	return results, nil
}

// getSymbolFromID gets symbol from token ID using reverse lookup
func (a *PriceRepository) getSymbolFromID(tokenID string) string {
	// Reverse lookup in symbolToID map
	for symbol, id := range a.symbolToID {
		if id == tokenID {
			return symbol
		}
	}
	// If not found, return uppercase first 3-4 chars of tokenID as fallback
	if len(tokenID) >= 4 {
		return strings.ToUpper(tokenID[:4])
	} else if len(tokenID) >= 3 {
		return strings.ToUpper(tokenID[:3])
	}
	return strings.ToUpper(tokenID)
}
