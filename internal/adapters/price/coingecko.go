package price

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"testtask/internal/domain/price"
)

// CoinGeckoProvider implements PriceProvider using CoinGecko API
type CoinGeckoProvider struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	symbolToID map[string]string // Cache for symbol to CoinGecko ID mapping
}

// NewCoinGeckoProvider creates a new CoinGecko provider
func NewCoinGeckoProvider(client *http.Client, apiKey string) *CoinGeckoProvider {
	return &CoinGeckoProvider{
		client:  client,
		baseURL: "https://api.coingecko.com/api/v3",
		apiKey:  apiKey,
	}
}

// CoinGeckoSimplePriceResponse represents the simple price response
type CoinGeckoSimplePriceResponse map[string]map[string]float64

// GetPrices retrieves prices for multiple tokens (batch request)
// Implements PriceProvider interface from domain
func (p *CoinGeckoProvider) GetPrices(
	ctx context.Context,
	tokens []*price.Token,
	currency string,
) (map[*price.Token]*price.Price, error) {
	if len(tokens) == 0 {
		return make(map[*price.Token]*price.Price), nil
	}

	// Normalize currency to lowercase for CoinGecko API
	currency = strings.ToLower(currency)
	if currency == "" {
		currency = "usd"
	}

	// Convert tokens to CoinGecko IDs
	tokenMap := make(map[string]*price.Token) // Map from CoinGecko ID to Token
	for _, token := range tokens {
		tokenMap[token.ID] = token
	}

	// CoinGecko API supports up to 250 IDs per request
	// Batch in chunks of 250
	const maxBatchSize = 250
	results := make(map[*price.Token]*price.Price)

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

		batchResults, err := p.fetchPricesBatch(ctx, batch, currency, tokenMap)
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
func (p *CoinGeckoProvider) fetchPricesBatch(
	ctx context.Context,
	tokenIDs []string,
	currency string,
	tokenMap map[string]*price.Token,
) (map[*price.Token]*price.Price, error) {
	idsParam := strings.Join(tokenIDs, ",")
	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s&include_last_updated_at=true&include_tokens=all&precision=8", p.baseURL, idsParam, currency)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CoinGecko API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var data CoinGeckoSimplePriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make(map[*price.Token]*price.Price)
	for tokenID, priceData := range data {
		if priceValue, ok := priceData[currency]; ok {
			// Get the original token from the map
			token, ok := tokenMap[tokenID]
			if !ok {
				// If token not in map, create one from the ID
				symbol := p.getSymbolFromID(tokenID)
				token = &price.Token{
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

			p := price.NewPrice(token, priceAmount, strings.ToUpper(currency))

			p.LastUpdated = lastUpdated
			results[token] = &p
		}
	}

	return results, nil
}

// getSymbolFromID gets symbol from token ID using reverse lookup
func (p *CoinGeckoProvider) getSymbolFromID(tokenID string) string {
	// Reverse lookup in symbolToID map
	for symbol, id := range p.symbolToID {
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
