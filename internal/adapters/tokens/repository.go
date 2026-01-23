package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"testtask/internal/domain/price"
)

// Repository loads tokens from coins.json and provides token lookup functionality
type Repository struct {
	mu *sync.RWMutex
	// Maps for fast lookups
	tokensByID      map[string]*TokenData   // CoinGecko ID -> TokenData
	tokensBySymbol  map[string][]*TokenData // Symbol (uppercase) -> []TokenData (multiple tokens can have same symbol)
	tokensByAddress map[string]*TokenData   // Ethereum address (lowercase) -> TokenData
	tokens          []*TokenData            // All tokens
}

// TokenData represents a token from coins.json
type TokenData struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Address   string            `json:"-"` // Ethereum address, populated from platforms.ethereum
	Platforms map[string]string `json:"platforms"`
}

// NewRepository creates a new tokens repository and loads tokens from coins.json
func NewRepository(filePath string) (*Repository, error) {
	repo := &Repository{
		tokensByID:      make(map[string]*TokenData),
		tokensBySymbol:  make(map[string][]*TokenData),
		tokensByAddress: make(map[string]*TokenData),
		tokens:          make([]*TokenData, 0),
		mu:              &sync.RWMutex{},
	}

	if err := repo.loadTokens(filePath); err != nil {
		return nil, fmt.Errorf("failed to load tokens: %w", err)
	}

	return repo, nil
}

// loadTokens loads tokens from the JSON file into memory
func (r *Repository) loadTokens(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var tokens []TokenData
	if err := json.Unmarshal(data, &tokens); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Build lookup maps
	for i := range tokens {
		token := &TokenData{
			ID:        tokens[i].ID,
			Symbol:    tokens[i].Symbol,
			Name:      tokens[i].Name,
			Platforms: make(map[string]string),
		}
		// Copy platforms map and extract Ethereum address
		for k, v := range tokens[i].Platforms {
			token.Platforms[k] = v
			// Extract Ethereum address if available
			if k == "ethereum" && v != "" {
				token.Address = v
			}
		}
		r.tokens = append(r.tokens, token)

		// Index by ID (lowercase for case-insensitive lookup)
		idLower := strings.ToLower(token.ID)
		r.tokensByID[idLower] = token

		// Index by symbol (uppercase for case-insensitive lookup)
		symbolUpper := strings.ToUpper(token.Symbol)
		r.tokensBySymbol[symbolUpper] = append(r.tokensBySymbol[symbolUpper], token)

		// Index by Ethereum address if available
		if token.Address != "" {
			addrLower := strings.ToLower(token.Address)
			r.tokensByAddress[addrLower] = token
		}
	}

	return nil
}

// IsSupported checks if a token is supported by ID, symbol, or Ethereum address
// It performs case-insensitive matching
func (r *Repository) IsSupported(identifier string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return false
	}

	// Check by ID (case-insensitive)
	idLower := strings.ToLower(identifier)
	if _, ok := r.tokensByID[idLower]; ok {
		return true
	}

	// Check by symbol (case-insensitive)
	symbolUpper := strings.ToUpper(identifier)
	if tokens, ok := r.tokensBySymbol[symbolUpper]; ok && len(tokens) > 0 {
		return true
	}

	// Check by Ethereum address (case-insensitive, with or without 0x prefix)
	addrLower := strings.ToLower(identifier)
	if !strings.HasPrefix(addrLower, "0x") {
		addrLower = "0x" + addrLower
	}
	if _, ok := r.tokensByAddress[addrLower]; ok {
		return true
	}

	return false
}

// GetTokenByID retrieves a token by its CoinGecko ID
func (r *Repository) GetTokenByID(_ context.Context, id string) (*price.Token, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	idLower := strings.ToLower(id)
	token, ok := r.tokensByID[idLower]
	return r.ToDomainToken(token), ok
}

// GetTokenByAddress retrieves a token by its Ethereum address
func (r *Repository) GetTokenByAddress(_ context.Context, address string) (*price.Token, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	addrLower := strings.ToLower(address)
	if !strings.HasPrefix(addrLower, "0x") {
		addrLower = "0x" + addrLower
	}
	token, ok := r.tokensByAddress[addrLower]

	return r.ToDomainToken(token), ok
}

// ToDomainToken converts TokenData to domain Token
func (r *Repository) ToDomainToken(token *TokenData) *price.Token {
	if token == nil {
		return nil
	}

	return &price.Token{
		ID:      token.ID,
		Symbol:  token.Symbol,
		Address: token.Address,
	}
}

// GetAllTokens returns all loaded tokens
func (r *Repository) GetAllTokens() []*TokenData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*TokenData, len(r.tokens))
	copy(result, r.tokens)
	return result
}

// Count returns the total number of loaded tokens
func (r *Repository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tokens)
}
