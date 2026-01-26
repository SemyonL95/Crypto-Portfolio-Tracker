package http

import (
	"math/big"
	portfolioService "testtask/internal/application/portfolio"
	"testtask/internal/domain/holding"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
	"testtask/internal/domain/transaction"
)

func ToHTTPTransaction(t *transaction.Transaction) *Transaction {
	if t == nil {
		return nil
	}
	return &Transaction{
		ID:           t.ID,
		Hash:         t.Hash,
		From:         t.From,
		To:           t.To,
		TokenAddress: t.TokenAddress,
		TokenSymbol:  t.TokenSymbol,
		Amount:       t.Amount,
		Type:         string(t.Type),
		Status:       string(t.Status),
		Method:       t.Method,
		Timestamp:    t.Timestamp,
		BlockNumber:  t.BlockNumber,
	}
}

func ToHTTPTransactions(transactions []transaction.Transaction) []Transaction {
	result := make([]Transaction, len(transactions))
	for i, t := range transactions {
		result[i] = *ToHTTPTransaction(&t)
	}
	return result
}

func ToDomainTransaction(t *Transaction) *transaction.Transaction {
	if t == nil {
		return nil
	}
	return &transaction.Transaction{
		ID:           t.ID,
		Hash:         t.Hash,
		From:         t.From,
		To:           t.To,
		TokenAddress: t.TokenAddress,
		TokenSymbol:  t.TokenSymbol,
		Amount:       t.Amount,
		Type:         transaction.TransactionType(t.Type),
		Status:       transaction.TransactionStatus(t.Status),
		Method:       t.Method,
		Timestamp:    t.Timestamp,
		BlockNumber:  t.BlockNumber,
	}
}

func ToHTTPHolding(h *holding.Holding) *Holding {
	if h == nil {
		return nil
	}
	return &Holding{
		ID:           h.ID,
		TokenAddress: h.Token.Address,
		TokenSymbol:  h.Token.Symbol,
		Amount:       h.Amount,
		//ValueUSD:     0,
		//PriceUSD:     0,
		UpdatedAt: h.UpdatedAt,
	}
}

// ToHTTPHoldings converts a slice of holding Holding to HTTP Holding
func ToHTTPHoldings(holdings []*holding.Holding) []*Holding {
	result := make([]*Holding, len(holdings))
	for i, h := range holdings {
		result[i] = ToHTTPHolding(h)
	}
	return result
}

func ToHTTPHoldingsWithPrices(holdings []*holding.Holding, prices map[*token.Token]*price.Price) []*Holding {
	result := make([]*Holding, len(holdings))
	for i, h := range holdings {
		result[i] = ToHTTPHoldingWithPrice(h, prices)
	}
	return result
}

func ToHTTPHoldingWithPrice(h *holding.Holding, prices map[*token.Token]*price.Price) *Holding {
	holding := ToHTTPHolding(h)
	if h != nil && h.Token != nil && prices != nil {
		if priceData, ok := prices[h.Token]; ok && priceData != nil {
			// Convert coingecko to big.Int (coingecko is stored as big.Int in cents or smallest unit)
			holding.PriceUSD = priceData.Value
			// Calculate value: amount * coingecko
			if h.Amount != nil && priceData.Value != nil {
				holding.ValueUSD = new(big.Int).Mul(h.Amount, priceData.Value)
			}
		}
	}
	return holding
}

func ToDomainHolding(h *Holding) *holding.Holding {
	if h == nil {
		return nil
	}
	return &holding.Holding{
		//Token:     ToDomainToken(),
		ID:        h.ID,
		Amount:    h.Amount,
		UpdatedAt: h.UpdatedAt,
	}
}

func ToHTTPPortfolio(p *portfolio.Portfolio, prices map[*token.Token]*price.Price) *Portfolio {
	if p == nil {
		return nil
	}
	holdings := ToHTTPHoldingsWithPrices(p.Holdings, prices)
	return &Portfolio{
		ID:        p.ID,
		Holdings:  holdings,
		UpdatedAt: p.UpdatedAt,
	}
}

func ToDomainPortfolio(p *Portfolio) *portfolio.Portfolio {
	if p == nil {
		return nil
	}
	holdings := make([]*holding.Holding, len(p.Holdings))
	for i, h := range p.Holdings {
		holdings[i] = ToDomainHolding(h)
	}
	return &portfolio.Portfolio{
		ID:        p.ID,
		Holdings:  holdings,
		UpdatedAt: p.UpdatedAt,
	}
}

func ToDomainToken(id, addr, symbol string) *token.Token {
	return &token.Token{
		ID:      id,
		Symbol:  symbol,
		Address: addr,
	}
}

// ToHTTPPortfolioAssets converts service PortfolioAssets to HTTP PortfolioAssets
func ToHTTPPortfolioAssets(pa *portfolioService.PortfolioAssets) *PortfolioAssets {
	if pa == nil {
		return nil
	}

	assets := make([]*Asset, len(pa.Assets))
	for i, asset := range pa.Assets {
		assets[i] = ToHTTPAsset(asset)
	}

	return &PortfolioAssets{
		PortfolioID:  pa.PortfolioID,
		Address:      pa.Address,
		Assets:       assets,
		TotalValue:   pa.TotalValue,
		Currency:     pa.Currency,
		CalculatedAt: pa.CalculatedAt,
	}
}

// ToHTTPAsset converts service Asset to HTTP Asset
func ToHTTPAsset(a *portfolioService.Asset) *Asset {
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

	var priceUSD *big.Int
	if a.Price != nil && a.Price.Value != nil {
		priceUSD = new(big.Int).Set(a.Price.Value)
	}

	return &Asset{
		Token:    tokenInfo,
		Amount:   a.Amount,
		PriceUSD: priceUSD,
		ValueUSD: a.Value,
		Source:   a.Source,
	}
}
