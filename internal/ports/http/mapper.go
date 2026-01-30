package http

import (
	"math"
	"math/big"

	domainHolding "testtask/internal/domain/holding"
	domainPortfolio "testtask/internal/domain/portfolio"
	"testtask/internal/domain/transaction"
)

// bigIntToUSDFloat converts a big.Int with 8 decimal places to a float64
func bigIntToUSDFloat(value *big.Int) float64 {
	if value == nil {
		return 0
	}

	// USD has 8 decimal places, so divide by 10^8
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil))
	valueBigFloat := new(big.Float).SetInt(value)
	result := new(big.Float).Quo(valueBigFloat, divisor)

	floatVal, _ := result.Float64()
	// Round to 8 decimal places
	return math.Round(floatVal*100000000) / 100000000
}

func ToHTTPTransaction(t *transaction.Transaction) *Transaction {
	if t == nil {
		return nil
	}

	// Convert big.Int amount to string for JSON serialization
	amountStr := "0"
	if t.Amount != nil {
		amountStr = t.Amount.String()
	}

	return &Transaction{
		ID:           t.ID,
		Hash:         t.Hash,
		From:         t.From,
		To:           t.To,
		TokenAddress: t.TokenAddress,
		TokenSymbol:  t.TokenSymbol,
		Amount:       amountStr,
		Type:         string(t.Type),
		Status:       string(t.Status),
		Direction:    string(t.Direction),
		Method:       t.Method,
		Timestamp:    t.Timestamp,
		BlockNumber:  t.BlockNumber,
	}
}

func ToHTTPTransactions(transactions transaction.Transactions) []Transaction {
	result := make([]Transaction, len(transactions))
	for i, t := range transactions {
		if t == nil {
			continue
		}
		result[i] = *ToHTTPTransaction(t)
	}
	return result
}

// ToHTTPTransactionsFromSlice converts a slice of transaction values to HTTP transactions.
func ToHTTPTransactionsFromSlice(transactions []transaction.Transaction) []Transaction {
	result := make([]Transaction, len(transactions))
	for i := range transactions {
		result[i] = *ToHTTPTransaction(&transactions[i])
	}
	return result
}

func ToDomainTransaction(t *Transaction) *transaction.Transaction {
	if t == nil {
		return nil
	}

	// Parse string amount to big.Int
	amount := big.NewInt(0)
	if t.Amount != "" {
		if parsed, ok := new(big.Int).SetString(t.Amount, 10); ok {
			amount = parsed
		}
	}

	return &transaction.Transaction{
		ID:           t.ID,
		Hash:         t.Hash,
		From:         t.From,
		To:           t.To,
		TokenAddress: t.TokenAddress,
		TokenSymbol:  t.TokenSymbol,
		Amount:       amount,
		Type:         transaction.TransactionType(t.Type),
		Status:       transaction.TransactionStatus(t.Status),
		Direction:    transaction.TransactionDirection(t.Direction),
		Method:       t.Method,
		Timestamp:    t.Timestamp,
		BlockNumber:  t.BlockNumber,
	}
}

func ToHTTPHolding(h *domainHolding.Holding) *Holding {
	if h == nil {
		return nil
	}
	return &Holding{
		ID:           h.ID,
		TokenAddress: h.Token.Address,
		TokenSymbol:  h.Token.Symbol,
		Amount:       h.Amount,
	}
}

// ToHTTPHoldings converts a slice of holding Holding to HTTP Holding
func ToHTTPHoldings(holdings []*domainHolding.Holding) []*Holding {
	result := make([]*Holding, len(holdings))
	for i, h := range holdings {
		result[i] = ToHTTPHolding(h)
	}
	return result
}

func ToDomainHolding(h *Holding) *domainHolding.Holding {
	if h == nil {
		return nil
	}
	return &domainHolding.Holding{
		ID:     h.ID,
		Amount: h.Amount,
	}
}

func ToHTTPPortfolios(p []*domainPortfolio.Portfolio) []*Portfolio {
	if p == nil {
		return nil
	}

	var result []*Portfolio
	for _, portf := range p {
		result = append(result, ToHTTPPortfolio(portf))
	}

	return result
}

func ToHTTPPortfolio(p *domainPortfolio.Portfolio) *Portfolio {
	if p == nil {
		return nil
	}
	holdings := ToHTTPHoldings(p.Holdings)
	return &Portfolio{
		ID:       p.ID,
		Holdings: holdings,
	}
}

// ToHTTPPortfolioAssets converts service PortfolioAssets to HTTP PortfolioAssets
func ToHTTPPortfolioAssets(pa *domainPortfolio.Portfolio, a []*domainPortfolio.Asset) *PortfolioAssets {
	if pa == nil {
		return nil
	}

	assets := make([]*Asset, len(a))
	for i, asset := range a {
		assets[i] = ToHTTPAsset(asset)
	}

	return &PortfolioAssets{
		PortfolioID: pa.ID,
		Address:     pa.Address,
		Assets:      assets,
	}
}

// ToHTTPAsset converts service Asset to HTTP Asset
func ToHTTPAsset(a *domainPortfolio.Asset) *Asset {
	if a == nil {
		return nil
	}

	var tokenInfo *TokenInfo
	if a.Token != nil {
		tokenInfo = &TokenInfo{
			ID:      a.Token.ID,
			Name:    a.Token.Name,
			Symbol:  a.Token.Symbol,
			Address: a.Token.Address,
			Decimal: a.Token.Decimal,
		}
	}

	var priceUSD float64
	if a.Price != nil && a.Price.Value != nil {
		priceUSD = bigIntToUSDFloat(a.Price.Value)
	}

	var valueUSD float64
	if a.Value != nil {
		valueUSD = bigIntToUSDFloat(a.Value)
	}

	return &Asset{
		Token:    tokenInfo,
		Amount:   a.Amount,
		ValueUSD: valueUSD,
		PriceUSD: priceUSD,
		Source:   a.Source,
	}
}
