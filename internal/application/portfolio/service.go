package portfolio

import (
	"context"
	"errors"
	"math/big"
	domainHolding "testtask/internal/domain/holding"
	domainPortfolio "testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
	domainTransaction "testtask/internal/domain/transaction"
	"time"
)

var (
	ErrInvalidHolding   = errors.New("invalid domainHolding")
	ErrInvalidPortfolio = errors.New("invalid portfolio")
	ErrPortfolioExists  = errors.New("portfolio already exists")
)

type Service struct {
	portfolioRepo   domainPortfolio.Repository
	holdingRepo     domainHolding.Repository
	transactionRepo domainTransaction.Repository
	priceProvider   price.Provider
}

func NewService(repo domainPortfolio.Repository, priceProvider price.Provider) *Service {
	return &Service{
		portfolioRepo: repo,
		priceProvider: priceProvider,
	}
}

// SetHoldingRepo sets the holding repository (for backward compatibility)
func (s *Service) SetHoldingRepo(holdingRepo domainHolding.Repository) {
	s.holdingRepo = holdingRepo
}

// SetTransactionRepo sets the transaction repository
func (s *Service) SetTransactionRepo(transactionRepo domainTransaction.Repository) {
	s.transactionRepo = transactionRepo
}

func (s *Service) CreatePortfolio(ctx context.Context, p *domainPortfolio.Portfolio) error {
	if p == nil {
		return ErrInvalidPortfolio
	}

	if p.Address == "" {
		return ErrInvalidPortfolio
	}

	existingByAddr, err := s.portfolioRepo.GetByAddress(ctx, p.Address)
	if err == nil && existingByAddr != nil {
		return domainPortfolio.ErrPortfolioAddressExists
	}
	if err != nil && !errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
		return err
	}

	if p.ID == "" {
		newPortfolio := domainPortfolio.NewPortfolio("", p.Address)
		return s.portfolioRepo.Create(ctx, newPortfolio)
	}

	existing, err := s.portfolioRepo.GetByID(ctx, p.ID)
	if err == nil && existing != nil {
		return ErrPortfolioExists
	}

	if err != nil && !errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
		return err
	}

	newPortfolio := domainPortfolio.NewPortfolio(p.ID, p.Address)
	return s.portfolioRepo.Create(ctx, newPortfolio)
}

func (s *Service) GetPortfolio(ctx context.Context, portfolioID string) (*domainPortfolio.Portfolio, error) {
	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (s *Service) GetHoldings(ctx context.Context, portfolioID string) ([]*domainHolding.Holding, error) {
	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	holdings := make([]*domainHolding.Holding, len(p.Holdings))
	for i, h := range p.Holdings {
		holdings[i] = h
	}
	return holdings, nil
}

func (s *Service) AddHolding(ctx context.Context, portfolioID string, holding *domainHolding.Holding) error {
	if holding.Token == nil {
		return ErrInvalidHolding
	}

	if holding.Amount == nil || holding.Amount.Sign() <= 0 {
		return ErrInvalidHolding
	}

	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			return domainPortfolio.ErrPortfolioNotFound
		}
		return err
	}

	existing := s.FindHoldingByToken(p, holding.Token.ID)
	if existing != nil {
		newAmount := new(big.Int).Add(existing.Amount, holding.Amount)
		s.UpdateAmount(existing, newAmount)

		return s.holdingRepo.UpdateHolding(ctx, portfolioID, existing)
	}

	// Create new domainHolding with UUID and portfolioID
	newHolding := domainHolding.NewHolding(portfolioID, "", holding.Token, holding.Amount)

	return s.holdingRepo.CreateHolding(ctx, portfolioID, newHolding)
}

// UpdateHolding updates an existing domainHolding
func (s *Service) UpdateHolding(ctx context.Context, portfolioID string, holdingID string, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return ErrInvalidHolding
	}

	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			return domainPortfolio.ErrPortfolioNotFound
		}
		return err
	}

	existing := s.FindHolding(p, holdingID)
	if existing == nil {
		return domainHolding.ErrHoldingNotFound
	}

	s.UpdateAmount(existing, amount)

	return s.holdingRepo.UpdateHolding(ctx, portfolioID, existing)
}

// DeleteHolding removes a domainHolding from the portfolio
func (s *Service) DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error {
	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			return domainPortfolio.ErrPortfolioNotFound
		}
		return err
	}

	holding := s.FindHolding(p, holdingID)
	if holding == nil {
		return domainHolding.ErrHoldingNotFound
	}

	return s.holdingRepo.DeleteHolding(ctx, portfolioID, holdingID)
}

func (s *Service) UpdateAmount(holding *domainHolding.Holding, amount *big.Int) {
	holding.Amount = new(big.Int).Set(amount)
	holding.UpdatedAt = time.Now()
}

func (s *Service) FindHolding(p *domainPortfolio.Portfolio, holdingID string) *domainHolding.Holding {
	for _, h := range p.Holdings {
		if h.ID == holdingID {
			return h
		}
	}
	return nil
}

func (s *Service) FindHoldingByToken(p *domainPortfolio.Portfolio, tokenID string) *domainHolding.Holding {
	for _, h := range p.Holdings {
		if h.Token != nil && h.Token.ID == tokenID {
			return h
		}
	}
	return nil
}

type PortfolioValue struct {
	Portfolio     *domainPortfolio.Portfolio
	TotalValue    *big.Int
	Currency      string
	HoldingValues []*HoldingValue
	CalculatedAt  time.Time
}

type HoldingValue struct {
	Holding *domainHolding.Holding
	Price   *price.Price
	Value   *big.Int
}

// Asset represents an asset from holdings or transactions with its value
type Asset struct {
	Token  *token.Token
	Amount *big.Int
	Price  *price.Price
	Value  *big.Int
	Source string // "holding" or "transaction"
}

// PortfolioAssets represents all assets in a portfolio with their values
type PortfolioAssets struct {
	PortfolioID  string
	Address      string
	Assets       []*Asset
	TotalValue   *big.Int
	Currency     string
	CalculatedAt time.Time
}

// GetPortfolioAssets retrieves all assets from holdings and transactions and calculates their values
func (s *Service) GetPortfolioAssets(ctx context.Context, portfolioID string, currency string) (*PortfolioAssets, error) {
	if currency == "" {
		currency = "usd"
	}

	// Get portfolio with holdings
	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// Get all transactions for the portfolio address
	var transactions []domainTransaction.Transaction
	if s.transactionRepo != nil && p.Address != "" {
		opts := domainTransaction.FilterOptions{
			Address: p.Address,
		}
		allTransactions, err := s.transactionRepo.GetAllTransactionsByAddress(ctx, p.Address, opts)
		if err == nil {
			transactions = allTransactions
		}
		// Ignore transaction errors - we'll just use holdings if transactions fail
	}

	// Collect unique tokens from holdings
	tokenMap := make(map[string]*token.Token)
	for _, holding := range p.Holdings {
		if holding.Token != nil {
			tokenMap[holding.Token.ID] = holding.Token
			// Also index by address for transaction matching
			if holding.Token.Address != "" {
				tokenMap[holding.Token.Address] = holding.Token
			}
		}
	}

	transactionAmounts := make(map[string]*big.Int) // token address -> total amount
	for _, tx := range transactions {
		if tx.TokenAddress == "" {
			continue
		}

		amount, ok := new(big.Int).SetString(tx.Amount, 10)
		if !ok {
			continue
		}

		// Determine if transaction is incoming or outgoing
		// For simplicity, we'll aggregate all transactions
		// In a real scenario, you'd want to track net balance changes
		if tx.Direction == domainTransaction.TransactionDirectionIn {
			if existing, ok := transactionAmounts[tx.TokenAddress]; ok {
				transactionAmounts[tx.TokenAddress] = new(big.Int).Add(existing, amount)
			} else {
				transactionAmounts[tx.TokenAddress] = new(big.Int).Set(amount)
			}
		} else if tx.Direction == domainTransaction.TransactionDirectionOut {
			if existing, ok := transactionAmounts[tx.TokenAddress]; ok {
				transactionAmounts[tx.TokenAddress] = new(big.Int).Sub(existing, amount)
			} else {
				transactionAmounts[tx.TokenAddress] = new(big.Int).Neg(amount)
			}
		}
	}

	// Convert token map to slice for price lookup
	tokens := make([]*token.Token, 0, len(tokenMap))
	for _, t := range tokenMap {
		tokens = append(tokens, t)
	}

	// Get prices for all tokens
	prices, err := s.priceProvider.GetPrices(ctx, tokens, currency)
	if err != nil {
		return nil, err
	}

	// Build asset list
	assets := make([]*Asset, 0)
	totalValue := big.NewInt(0)

	// Add assets from holdings
	for _, holding := range p.Holdings {
		if holding.Token == nil || holding.Amount == nil {
			continue
		}

		priceData, hasPrice := prices[holding.Token]
		if !hasPrice {
			priceData = nil
		}

		value := big.NewInt(0)
		if priceData != nil && priceData.Value != nil {
			value = domainPortfolio.CalculateValue(holding.Amount, holding.Token, priceData)
			if value != nil {
				totalValue.Add(totalValue, value)
			}
		}

		assets = append(assets, &Asset{
			Token:  holding.Token,
			Amount: new(big.Int).Set(holding.Amount),
			Price:  priceData,
			Value:  value,
			Source: "holding",
		})
	}

	// Add assets from transactions (only if not already in holdings)
	for tokenAddr, amount := range transactionAmounts {
		// Skip if amount is zero or negative
		if amount.Sign() <= 0 {
			continue
		}

		// Check if we already have this token in holdings
		alreadyInHoldings := false
		for _, holding := range p.Holdings {
			if holding.Token != nil && holding.Token.Address == tokenAddr {
				alreadyInHoldings = true
				break
			}
		}

		if alreadyInHoldings {
			continue
		}

		t, exists := tokenMap[tokenAddr]
		if !exists {
			continue
		}

		priceData, hasPrice := prices[t]
		if !hasPrice {
			priceData = nil
		}

		value := big.NewInt(0)
		if priceData != nil && priceData.Value != nil {
			value = domainPortfolio.CalculateValue(amount, t, priceData)
			if value != nil {
				totalValue.Add(totalValue, value)
			}
		}

		assets = append(assets, &Asset{
			Token:  t,
			Amount: new(big.Int).Set(amount),
			Price:  priceData,
			Value:  value,
			Source: "transaction",
		})
	}

	return &PortfolioAssets{
		PortfolioID:  portfolioID,
		Address:      p.Address,
		Assets:       assets,
		TotalValue:   totalValue,
		Currency:     currency,
		CalculatedAt: time.Now(),
	}, nil
}
