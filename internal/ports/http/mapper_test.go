package http

import (
	"math/big"
	"testing"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"testtask/internal/domain/transaction"
	"time"
)

func TestToHTTPTransaction(t *testing.T) {
	tx := &transaction.Transaction{
		ID:           "tx-1",
		Hash:         "0x123",
		From:         "0xabc",
		To:           "0xdef",
		TokenAddress: "0xtoken",
		TokenSymbol:  "ETH",
		Amount:       "100",
		Type:         transaction.TransactionTypeSend,
		Status:       transaction.TransactionStatusSuccess,
		Method:       "transfer",
		Timestamp:    time.Now(),
		BlockNumber:  12345,
	}

	result := ToHTTPTransaction(tx)
	if result == nil {
		t.Fatal("ToHTTPTransaction() returned nil")
	}
	if result.ID != tx.ID {
		t.Errorf("ToHTTPTransaction() ID = %v, want %v", result.ID, tx.ID)
	}
	if result.Hash != tx.Hash {
		t.Errorf("ToHTTPTransaction() Hash = %v, want %v", result.Hash, tx.Hash)
	}
	if result.Type != string(tx.Type) {
		t.Errorf("ToHTTPTransaction() Type = %v, want %v", result.Type, tx.Type)
	}
}

func TestToHTTPTransaction_Nil(t *testing.T) {
	result := ToHTTPTransaction(nil)
	if result != nil {
		t.Errorf("ToHTTPTransaction(nil) = %v, want nil", result)
	}
}

func TestToHTTPTransactions(t *testing.T) {
	txs := []transaction.Transaction{
		{
			ID:   "tx-1",
			Hash: "0x123",
		},
		{
			ID:   "tx-2",
			Hash: "0x456",
		},
	}

	result := ToHTTPTransactions(txs)
	if len(result) != 2 {
		t.Errorf("ToHTTPTransactions() returned %d items, want 2", len(result))
	}
	if result[0].ID != "tx-1" {
		t.Errorf("ToHTTPTransactions() first ID = %v, want tx-1", result[0].ID)
	}
}

func TestToHTTPHolding(t *testing.T) {
	token := &price.Token{
		ID:      "bitcoin",
		Symbol:  "BTC",
		Address: "0xbtc",
	}
	holding := &portfolio.Holding{
		ID:        "holding-1",
		Token:     token,
		Amount:    big.NewInt(100),
		UpdatedAt: time.Now(),
	}

	result := ToHTTPHolding(holding)
	if result == nil {
		t.Fatal("ToHTTPHolding() returned nil")
	}
	if result.ID != holding.ID {
		t.Errorf("ToHTTPHolding() ID = %v, want %v", result.ID, holding.ID)
	}
	if result.TokenAddress != token.Address {
		t.Errorf("ToHTTPHolding() TokenAddress = %v, want %v", result.TokenAddress, token.Address)
	}
	if result.Amount.Cmp(holding.Amount) != 0 {
		t.Errorf("ToHTTPHolding() Amount = %v, want %v", result.Amount, holding.Amount)
	}
}

func TestToHTTPHolding_Nil(t *testing.T) {
	result := ToHTTPHolding(nil)
	if result != nil {
		t.Errorf("ToHTTPHolding(nil) = %v, want nil", result)
	}
}

func TestToHTTPHoldings(t *testing.T) {
	token := &price.Token{
		ID:     "bitcoin",
		Symbol: "BTC",
	}
	holdings := []*portfolio.Holding{
		{
			ID:     "holding-1",
			Token:  token,
			Amount: big.NewInt(100),
		},
		{
			ID:     "holding-2",
			Token:  token,
			Amount: big.NewInt(200),
		},
	}

	result := ToHTTPHoldings(holdings)
	if len(result) != 2 {
		t.Errorf("ToHTTPHoldings() returned %d items, want 2", len(result))
	}
	if result[0].ID != "holding-1" {
		t.Errorf("ToHTTPHoldings() first ID = %v, want holding-1", result[0].ID)
	}
}

func TestToHTTPHoldingWithPrice(t *testing.T) {
	token := &price.Token{
		ID:      "bitcoin",
		Symbol:  "BTC",
		Address: "0xbtc",
	}
	holding := &portfolio.Holding{
		ID:     "holding-1",
		Token:  token,
		Amount: big.NewInt(100),
	}

	priceData := &price.Price{
		Token: token,
		Value: big.NewInt(50000), // $50,000 per BTC
	}

	prices := map[*price.Token]*price.Price{
		token: priceData,
	}

	result := ToHTTPHoldingWithPrice(holding, prices)
	if result == nil {
		t.Fatal("ToHTTPHoldingWithPrice() returned nil")
	}
	if result.PriceUSD == nil || result.PriceUSD.Cmp(priceData.Value) != 0 {
		t.Errorf("ToHTTPHoldingWithPrice() PriceUSD = %v, want %v", result.PriceUSD, priceData.Value)
	}
	expectedValue := big.NewInt(5000000) // 100 * 50000
	if result.ValueUSD == nil || result.ValueUSD.Cmp(expectedValue) != 0 {
		t.Errorf("ToHTTPHoldingWithPrice() ValueUSD = %v, want %v", result.ValueUSD, expectedValue)
	}
}

func TestToHTTPPortfolio(t *testing.T) {
	token := &price.Token{
		ID:      "bitcoin",
		Symbol:  "BTC",
		Address: "0xbtc",
	}
	portfolio := &portfolio.Portfolio{
		ID: "portfolio-1",
		Holdings: []*portfolio.Holding{
			{
				ID:     "holding-1",
				Token:  token,
				Amount: big.NewInt(100),
			},
		},
		UpdatedAt: time.Now(),
	}

	priceData := &price.Price{
		Token: token,
		Value: big.NewInt(50000),
	}
	prices := map[*price.Token]*price.Price{
		token: priceData,
	}

	result := ToHTTPPortfolio(portfolio, prices)
	if result == nil {
		t.Fatal("ToHTTPPortfolio() returned nil")
	}
	if result.ID != portfolio.ID {
		t.Errorf("ToHTTPPortfolio() ID = %v, want %v", result.ID, portfolio.ID)
	}
	if len(result.Holdings) != 1 {
		t.Errorf("ToHTTPPortfolio() Holdings length = %v, want 1", len(result.Holdings))
	}
	if result.Holdings[0].ValueUSD == nil {
		t.Error("ToHTTPPortfolio() Holdings[0].ValueUSD is nil, expected calculated value")
	}
}

func TestToHTTPPortfolio_Nil(t *testing.T) {
	result := ToHTTPPortfolio(nil, nil)
	if result != nil {
		t.Errorf("ToHTTPPortfolio(nil, nil) = %v, want nil", result)
	}
}

func TestToDomainTransaction(t *testing.T) {
	httpTx := &Transaction{
		ID:           "tx-1",
		Hash:         "0x123",
		From:         "0xabc",
		To:           "0xdef",
		TokenAddress: "0xtoken",
		TokenSymbol:  "ETH",
		Amount:       "100",
		Type:         "send",
		Status:       "success",
		Method:       "transfer",
		Timestamp:    time.Now(),
		BlockNumber:  12345,
	}

	result := ToDomainTransaction(httpTx)
	if result == nil {
		t.Fatal("ToDomainTransaction() returned nil")
	}
	if result.ID != httpTx.ID {
		t.Errorf("ToDomainTransaction() ID = %v, want %v", result.ID, httpTx.ID)
	}
	if result.Type != transaction.TransactionType(httpTx.Type) {
		t.Errorf("ToDomainTransaction() Type = %v, want %v", result.Type, httpTx.Type)
	}
}

func TestToDomainTransaction_Nil(t *testing.T) {
	result := ToDomainTransaction(nil)
	if result != nil {
		t.Errorf("ToDomainTransaction(nil) = %v, want nil", result)
	}
}
