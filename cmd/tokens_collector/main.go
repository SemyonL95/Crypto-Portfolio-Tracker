package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"testtask/internal/adapters/coingecko"

	"github.com/joho/godotenv"
)

// Token represents the merged token data
type Token struct {
	ID      string `json:"ID"`
	Name    string `json:"Name"`
	Symbol  string `json:"Symbol"`
	Address string `json:"Address"`
	Decimal int    `json:"Decimal"`
	ChainID string `json:"ChainID,omitempty"`
}

// CoinGeckoToken represents a token from /coins/list endpoint
type CoinGeckoToken struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// EthereumToken represents a token from /token_lists/ethereum/all.json endpoint
type EthereumToken struct {
	ChainID  int    `json:"chainId"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	LogoURI  string `json:"logoURI,omitempty"`
}

// TokenListResponse represents the response from /token_lists/ethereum/all.json
type TokenListResponse struct {
	Name    string          `json:"name"`
	Tokens  []EthereumToken `json:"tokens"`
	Version struct {
		Major int `json:"major"`
		Minor int `json:"minor"`
		Patch int `json:"patch"`
	} `json:"version"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	apiKey := os.Getenv("COINGECKO_API_KEY")
	if apiKey == "" {
		log.Fatal("COINGECKO_API_KEY environment variable is required")
	}

	baseURL := os.Getenv("COINGECKO_BASE_URL")
	if baseURL == "" {
		baseURL = "https://pro-api.coingecko.com/api/v3"
	}

	ctx := context.Background()

	// Create CoinGecko client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	client := coingecko.NewClient(httpClient, baseURL, apiKey)

	log.Println("Fetching token list from Ethereum...")
	var tokenListResp TokenListResponse
	if err := client.Get(ctx, "token_lists/ethereum/all.json", &tokenListResp); err != nil {
		log.Fatalf("Failed to fetch Ethereum token list: %v", err)
	}
	log.Printf("Fetched %d Ethereum tokens", len(tokenListResp.Tokens))

	log.Println("Fetching coins list from CoinGecko...")
	var coinsList []CoinGeckoToken
	if err := client.Get(ctx, "coins/list", &coinsList); err != nil {
		log.Fatalf("Failed to fetch coins list: %v", err)
	}
	log.Printf("Fetched %d coins from CoinGecko", len(coinsList))

	// Create a map of CoinGecko tokens by symbol for quick lookup
	coinsMap := make(map[string]CoinGeckoToken)
	for _, coin := range coinsList {
		// Use symbol as key (lowercase for case-insensitive matching)
		key := fmt.Sprintf("%s", coin.Symbol)
		coinsMap[key] = coin
	}

	// Merge the data
	tokens := make([]Token, 0, len(tokenListResp.Tokens))
	seenAddresses := make(map[string]bool)

	for _, ethToken := range tokenListResp.Tokens {
		// Skip if we've already seen this address
		addressKey := fmt.Sprintf("%s-%d", ethToken.Address, ethToken.ChainID)
		if seenAddresses[addressKey] {
			continue
		}
		seenAddresses[addressKey] = true

		token := Token{
			Address: ethToken.Address,
			Name:    ethToken.Name,
			Symbol:  ethToken.Symbol,
			Decimal: ethToken.Decimals,
			ChainID: fmt.Sprintf("%d", ethToken.ChainID),
		}

		// Try to find matching CoinGecko ID by symbol
		if cgToken, exists := coinsMap[ethToken.Symbol]; exists {
			token.ID = cgToken.ID
		} else {
			// If no match found, use symbol as ID (fallback)
			token.ID = ethToken.Symbol
		}

		tokens = append(tokens, token)
	}

	log.Printf("Merged into %d unique tokens", len(tokens))

	// Write to file
	outputPath := "./static/tokens.json"
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tokens); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}

	log.Printf("Successfully wrote %d tokens to %s", len(tokens), outputPath)
}
