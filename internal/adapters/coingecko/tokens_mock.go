package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"testtask/internal/adapters/cache"
	"testtask/internal/domain/token"
)

type MockTokenRepository struct {
	addressCache *cache.Cache[string, *token.Token]
	tokenList    []*token.Token
}

func NewMockTokenRepository(tokensPath string) (*MockTokenRepository, error) {
	tokens, err := loadTokensFromFile(tokensPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokens: %w", err)
	}

	addressCache := cache.NewCache[string, *token.Token](len(tokens))

	addressMap := make(map[string]*token.Token, len(tokens))
	for _, t := range tokens {
		addressMap[t.Address] = t
	}

	ctx := context.TODO()
	addressCache.SetBatch(ctx, addressMap)

	return &MockTokenRepository{
		addressCache: addressCache,
		tokenList:    tokens,
	}, nil
}

func (r *MockTokenRepository) GetList(ctx context.Context) ([]*token.Token, error) {
	return r.tokenList, nil
}

func (r *MockTokenRepository) GetByAddress(ctx context.Context, address string) (*token.Token, error) {
	t, ok := r.addressCache.Get(ctx, address)
	if !ok {
		return nil, fmt.Errorf("token not found for address: %s", address)
	}
	return t, nil
}

// loadTokensFromFile loads tokens from a JSON file
func loadTokensFromFile(path string) ([]*token.Token, error) {
	filePath := path
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		filePath = filepath.Join(wd, path)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tokens file: %w", err)
	}

	var jsonTokens []struct {
		ID      string `json:"ID"`
		Name    string `json:"Name"`
		Symbol  string `json:"Symbol"`
		Address string `json:"Address"`
		Decimal uint8  `json:"Decimal"`
	}

	if err := json.Unmarshal(data, &jsonTokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tokens JSON: %w", err)
	}

	tokens := make([]*token.Token, 0, len(jsonTokens))
	for _, jt := range jsonTokens {
		tokens = append(tokens, &token.Token{
			ID:      jt.ID,
			Name:    jt.Name,
			Symbol:  jt.Symbol,
			Address: jt.Address,
			Decimal: jt.Decimal,
		})
	}

	return tokens, nil
}
