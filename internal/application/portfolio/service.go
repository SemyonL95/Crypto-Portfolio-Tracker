package portfolio

import (
	"context"
	"errors"
	"math/big"
	"strings"

	loggeradapter "testtask/internal/adapters/logger"
	domainHolding "testtask/internal/domain/holding"
	domainPortfolio "testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"testtask/internal/domain/token"
	domainTransaction "testtask/internal/domain/transaction"
	"time"

	"go.uber.org/zap"
)

var (
	ErrInvalidHolding   = errors.New("invalid domainHolding")
	ErrInvalidPortfolio = errors.New("invalid portfolio")
	ErrPortfolioExists  = errors.New("portfolio already exists")
)

// WETHAddress is the Wrapped ETH address on Ethereum mainnet.
// Used for price lookups when dealing with native ETH.
const WETHAddress = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

type Service struct {
	portfolioRepo   domainPortfolio.Repository
	holdingRepo     domainHolding.Repository
	transactionRepo domainTransaction.Provider
	tokenRepo       token.Repository
	priceProvider   price.PriceProvider
	logger          *loggeradapter.Logger
}

func (s *Service) ListPortfolios(ctx context.Context) ([]*domainPortfolio.Portfolio, error) {
	s.logger.Info("Listing all portfolios")
	portfolios, err := s.portfolioRepo.List(ctx)
	if err != nil {
		s.logger.Error("Failed to list portfolios", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Successfully listed portfolios", zap.Int("count", len(portfolios)))
	return portfolios, nil
}

func NewService(repo domainPortfolio.Repository, holdingRepo domainHolding.Repository, transactionRepo domainTransaction.Provider, tokenRepo token.Repository, priceProvider price.PriceProvider, logger *loggeradapter.Logger) *Service {
	if logger == nil {
		logger = loggeradapter.NewNopLogger()
	}
	return &Service{
		portfolioRepo:   repo,
		priceProvider:   priceProvider,
		holdingRepo:     holdingRepo,
		transactionRepo: transactionRepo,
		tokenRepo:       tokenRepo,
		logger:          logger,
	}
}

func (s *Service) CreatePortfolio(ctx context.Context, p *domainPortfolio.Portfolio) error {
	if p == nil {
		s.logger.Warn("Attempted to create portfolio with nil portfolio")
		return ErrInvalidPortfolio
	}

	if p.Address == "" {
		s.logger.Warn("Attempted to create portfolio with empty address")
		return ErrInvalidPortfolio
	}

	s.logger.Info("Creating portfolio", zap.String("address", p.Address), zap.String("id", p.ID))

	existingByAddr, err := s.portfolioRepo.GetByAddress(ctx, p.Address)
	if err == nil && existingByAddr != nil {
		s.logger.Warn("Portfolio with address already exists", zap.String("address", p.Address))
		return domainPortfolio.ErrPortfolioAddressExists
	}
	if err != nil && !errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
		s.logger.Error("Error checking existing portfolio by address", zap.String("address", p.Address), zap.Error(err))
		return err
	}

	if p.ID == "" {
		newPortfolio := domainPortfolio.NewPortfolio("", p.Address)
		if err := s.portfolioRepo.Create(ctx, newPortfolio); err != nil {
			s.logger.Error("Failed to create portfolio", zap.String("address", p.Address), zap.Error(err))
			return err
		}
		s.logger.Info("Successfully created portfolio", zap.String("address", p.Address), zap.String("id", newPortfolio.ID))
		return nil
	}

	existing, err := s.portfolioRepo.GetByID(ctx, p.ID)
	if err == nil && existing != nil {
		s.logger.Warn("Portfolio with ID already exists", zap.String("id", p.ID))
		return ErrPortfolioExists
	}

	if err != nil && !errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
		s.logger.Error("Error checking existing portfolio by ID", zap.String("id", p.ID), zap.Error(err))
		return err
	}

	newPortfolio := domainPortfolio.NewPortfolio(p.ID, p.Address)
	if err := s.portfolioRepo.Create(ctx, newPortfolio); err != nil {
		s.logger.Error("Failed to create portfolio", zap.String("id", p.ID), zap.String("address", p.Address), zap.Error(err))
		return err
	}
	s.logger.Info("Successfully created portfolio", zap.String("id", p.ID), zap.String("address", p.Address))
	return nil
}

func (s *Service) GetPortfolio(ctx context.Context, portfolioID string) (*domainPortfolio.Portfolio, error) {
	s.logger.Info("Getting portfolio", zap.String("portfolio_id", portfolioID))
	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		s.logger.Error("Failed to get portfolio", zap.String("portfolio_id", portfolioID), zap.Error(err))
		return nil, err
	}
	s.logger.Info("Successfully retrieved portfolio", zap.String("portfolio_id", portfolioID), zap.Int("holdings_count", len(p.Holdings)))
	return p, nil
}

func (s *Service) GetHoldings(ctx context.Context, portfolioID string) ([]*domainHolding.Holding, error) {
	s.logger.Info("Getting holdings", zap.String("portfolio_id", portfolioID))
	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		s.logger.Error("Failed to get holdings", zap.String("portfolio_id", portfolioID), zap.Error(err))
		return nil, err
	}
	holdings := make([]*domainHolding.Holding, len(p.Holdings))
	for i, h := range p.Holdings {
		holdings[i] = h
	}
	s.logger.Info("Successfully retrieved holdings", zap.String("portfolio_id", portfolioID), zap.Int("count", len(holdings)))
	return holdings, nil
}

func (s *Service) AddHolding(ctx context.Context, portfolioID string, holding *domainHolding.Holding) error {
	if holding.Token == nil {
		s.logger.Warn("Attempted to add holding with nil token", zap.String("portfolio_id", portfolioID))
		return ErrInvalidHolding
	}

	if holding.Amount == nil || holding.Amount.Sign() <= 0 {
		s.logger.Warn("Attempted to add holding with invalid amount", zap.String("portfolio_id", portfolioID), zap.String("token_id", holding.Token.ID))
		return ErrInvalidHolding
	}

	s.logger.Info("Adding holding", zap.String("portfolio_id", portfolioID), zap.String("token_id", holding.Token.ID), zap.String("amount", holding.Amount.String()))

	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			s.logger.Warn("Portfolio not found when adding holding", zap.String("portfolio_id", portfolioID))
			return domainPortfolio.ErrPortfolioNotFound
		}
		s.logger.Error("Failed to get portfolio when adding holding", zap.String("portfolio_id", portfolioID), zap.Error(err))
		return err
	}

	existing := s.FindHoldingByToken(p, holding.Token.ID)
	if existing != nil {
		s.logger.Info("Holding exists, updating amount", zap.String("portfolio_id", portfolioID), zap.String("token_id", holding.Token.ID), zap.String("holding_id", existing.ID))
		newAmount := new(big.Int).Add(existing.Amount, holding.Amount)
		s.UpdateAmount(existing, newAmount)

		if err := s.holdingRepo.UpdateHolding(ctx, portfolioID, existing); err != nil {
			s.logger.Error("Failed to update holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", existing.ID), zap.Error(err))
			return err
		}
		s.logger.Info("Successfully updated holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", existing.ID), zap.String("new_amount", newAmount.String()))
		return nil
	}

	// Create new domainHolding with UUID and portfolioID
	newHolding := domainHolding.NewHolding(portfolioID, "", holding.Token, holding.Amount)

	if err := s.holdingRepo.CreateHolding(ctx, portfolioID, newHolding); err != nil {
		s.logger.Error("Failed to create holding", zap.String("portfolio_id", portfolioID), zap.String("token_id", holding.Token.ID), zap.Error(err))
		return err
	}
	s.logger.Info("Successfully created holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", newHolding.ID), zap.String("token_id", holding.Token.ID))
	return nil
}

func (s *Service) UpdateHolding(ctx context.Context, portfolioID string, holdingID string, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		s.logger.Warn("Attempted to update holding with invalid amount", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID))
		return ErrInvalidHolding
	}

	s.logger.Info("Updating holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID), zap.String("amount", amount.String()))

	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			s.logger.Warn("Portfolio not found when updating holding", zap.String("portfolio_id", portfolioID))
			return domainPortfolio.ErrPortfolioNotFound
		}
		s.logger.Error("Failed to get portfolio when updating holding", zap.String("portfolio_id", portfolioID), zap.Error(err))
		return err
	}

	existing := s.FindHolding(p, holdingID)
	if existing == nil {
		s.logger.Warn("Holding not found", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID))
		return domainHolding.ErrHoldingNotFound
	}

	s.UpdateAmount(existing, amount)

	if err := s.holdingRepo.UpdateHolding(ctx, portfolioID, existing); err != nil {
		s.logger.Error("Failed to update holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID), zap.Error(err))
		return err
	}
	s.logger.Info("Successfully updated holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID), zap.String("amount", amount.String()))
	return nil
}

func (s *Service) DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error {
	s.logger.Info("Deleting holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID))

	p, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			s.logger.Warn("Portfolio not found when deleting holding", zap.String("portfolio_id", portfolioID))
			return domainPortfolio.ErrPortfolioNotFound
		}
		s.logger.Error("Failed to get portfolio when deleting holding", zap.String("portfolio_id", portfolioID), zap.Error(err))
		return err
	}

	holding := s.FindHolding(p, holdingID)
	if holding == nil {
		s.logger.Warn("Holding not found", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID))
		return domainHolding.ErrHoldingNotFound
	}

	if err := s.holdingRepo.DeleteHolding(ctx, portfolioID, holdingID); err != nil {
		s.logger.Error("Failed to delete holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID), zap.Error(err))
		return err
	}
	s.logger.Info("Successfully deleted holding", zap.String("portfolio_id", portfolioID), zap.String("holding_id", holdingID))
	return nil
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

func (s *Service) GetPortfolioAssets(ctx context.Context, portfolioID string, currency string) (*domainPortfolio.Portfolio, []*domainPortfolio.Asset, error) {
	s.logger.Info("Getting portfolio assets", zap.String("portfolio_id", portfolioID), zap.String("currency", currency))

	// Step 1: Fetch portfolio and holdings
	portfolio, err := s.portfolioRepo.GetByIDWithHoldings(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, domainPortfolio.ErrPortfolioNotFound) {
			s.logger.Warn("Portfolio not found", zap.String("portfolio_id", portfolioID))
			return nil, nil, domainPortfolio.ErrPortfolioNotFound
		}
		s.logger.Error("Failed to get portfolio", zap.String("portfolio_id", portfolioID), zap.Error(err))
		return nil, nil, err
	}
	s.logger.Info("Retrieved portfolio", zap.String("address", portfolio.Address), zap.Int("holdings_count", len(portfolio.Holdings)))

	opts := domainTransaction.FilterOptions{
		Address:  portfolio.Address,
		Page:     1,
		PageSize: 1000, // Fetch all transactions
	}

	tokenTxs, err := s.transactionRepo.TokenTxsByAddress(ctx, portfolio.Address, opts)
	if err != nil {
		s.logger.Warn("Failed to fetch token transactions, continuing with holdings only", zap.String("address", portfolio.Address), zap.Error(err))
		tokenTxs = []*domainTransaction.Transaction{}
	} else {
		s.logger.Debug("Fetched token transactions", zap.Int("count", len(tokenTxs)))
	}

	internalTxs, err := s.transactionRepo.InternalTxsByAddress(ctx, portfolio.Address, opts)
	if err != nil {
		s.logger.Warn("Failed to fetch internal transactions, continuing with holdings only", zap.String("address", portfolio.Address), zap.Error(err))
		internalTxs = []*domainTransaction.Transaction{}
	} else {
		s.logger.Debug("Fetched internal transactions", zap.Int("count", len(internalTxs)))
	}

	var allTransactions domainTransaction.Transactions
	allTransactions = append(allTransactions, tokenTxs...)
	allTransactions = append(allTransactions, internalTxs...)

	for _, tx := range allTransactions {
		tx.SetDirectionForAddress(portfolio.Address)
	}
	s.logger.Info("Combined all transactions", zap.Int("total_count", len(allTransactions)))

	txBalances, err := allTransactions.CalculateTokensAmounts()
	if err != nil {
		s.logger.Error("Failed to calculate transaction balances", zap.Error(err))
		return nil, nil, err
	}
	s.logger.Debug("Calculated transaction balances", zap.Int("token_count", len(txBalances)))

	type balanceProvider interface {
	}

	var ethBalance *big.Int
	ethBalance, err = s.transactionRepo.GetNativeBalance(ctx, portfolio.Address)
	if err != nil {
		s.logger.Debug("Failed to get native ETH balance, skipping on-chain balance", zap.Error(err))
		// Don't use transaction-calculated balance as it may be wrong without full transaction history
		// Instead, rely only on holdings data
		ethBalance = nil
	} else {
		s.logger.Debug("Fetched on-chain ETH balance", zap.String("balance", ethBalance.String()))
	}

	aggregatedBalances := make(map[string]*big.Int) // key: lowercase token address, "" for ETH

	for _, holding := range portfolio.Holdings {
		if holding.Token == nil || holding.Amount == nil {
			continue
		}
		tokenAddr := strings.ToLower(holding.Token.Address)
		// Treat zero address as native ETH (empty string key)
		if tokenAddr == token.ZeroAddress {
			tokenAddr = token.ZeroAddress
		}

		if existing, exists := aggregatedBalances[tokenAddr]; exists {
			aggregatedBalances[tokenAddr] = new(big.Int).Add(existing, holding.Amount)
		} else {
			aggregatedBalances[tokenAddr] = new(big.Int).Set(holding.Amount)
		}
	}
	s.logger.Debug("Added holdings to aggregated balances", zap.Int("holdings_count", len(portfolio.Holdings)))

	for tokenAddr, txBalance := range txBalances {
		if tokenAddr == "" {
			continue
		}

		if existing, exists := aggregatedBalances[tokenAddr]; exists {
			aggregatedBalances[tokenAddr] = new(big.Int).Add(existing, txBalance)
		} else {
			aggregatedBalances[tokenAddr] = new(big.Int).Set(txBalance)
		}
	}

	if ethBalance != nil && ethBalance.Sign() > 0 {
		aggregatedBalances[token.ZeroAddress] = new(big.Int).Set(ethBalance)
	}
	s.logger.Debug("Aggregated all balances", zap.Int("total_tokens", len(aggregatedBalances)))

	filteredBalances := make(map[string]*big.Int)
	for tokenAddr, balance := range aggregatedBalances {
		if balance != nil && balance.Sign() > 0 {
			filteredBalances[tokenAddr] = balance
		}
	}
	s.logger.Info("Filtered balances", zap.Int("non_zero_count", len(filteredBalances)))

	tokenAddresses := make([]string, 0, len(filteredBalances))
	for tokenAddr := range filteredBalances {
		if tokenAddr != "" {
			tokenAddresses = append(tokenAddresses, tokenAddr)
		}
	}

	// Fetch token metadata
	tokensMap := s.tokenRepo.GetByAddresses(ctx, tokenAddresses)
	s.logger.Debug("Fetched token metadata", zap.Int("found_count", len(tokensMap)))

	// Step 7: Build token list for price fetching
	var tokensForPricing []*token.Token
	tokenAddressToToken := make(map[string]*token.Token)

	// Add ERC-20 tokens
	for tokenAddr, tok := range tokensMap {
		if tok != nil {
			tokensForPricing = append(tokensForPricing, tok)
			tokenAddressToToken[strings.ToLower(tokenAddr)] = tok
		}
	}

	// Handle native ETH - use WETH for price lookup
	if _, hasETH := filteredBalances[token.ZeroAddress]; hasETH {
		// Create a token struct for ETH using WETH address for price lookup
		ethToken := &token.Token{
			ID:      "ethereum",
			Name:    "Ethereum",
			Symbol:  "ETH",
			Address: WETHAddress, // Use WETH address for price lookup
			Decimal: 18,
		}
		tokensForPricing = append(tokensForPricing, ethToken)
		tokenAddressToToken[token.ZeroAddress] = ethToken
	}

	// Fetch prices
	pricesMap, err := s.priceProvider.GetPrices(ctx, tokensForPricing, currency)
	if err != nil {
		s.logger.Error("Failed to fetch prices", zap.Error(err))
		return nil, nil, err
	}
	s.logger.Info("Fetched prices", zap.Int("price_count", len(pricesMap)))

	// Step 8: Build Asset structs
	assets := make([]*domainPortfolio.Asset, 0, len(filteredBalances))

	for tokenAddr, balance := range filteredBalances {

		tok := tokenAddressToToken[tokenAddr]
		if tok == nil {
			s.logger.Warn("Token metadata not found, skipping", zap.String("address", tokenAddr))
			continue
		}

		if tok.Name == "ETH" {
			tok.Address = token.ZeroAddress
		}

		// Find price for this token
		var assetPrice *price.Price
		for priceToken, p := range pricesMap {
			// Match by address (case-insensitive)
			if strings.EqualFold(priceToken.Address, tok.Address) {
				assetPrice = p
				break
			}
		}

		if assetPrice == nil {
			s.logger.Warn("Price not found for token, skipping value calculation", zap.String("token", tok.Symbol), zap.String("address", tok.Address))
			// Still create asset but without price/value
			asset := &domainPortfolio.Asset{
				Token:  tok,
				Amount: balance,
				Price:  nil,
				Value:  nil,
				Source: "aggregated",
			}
			assets = append(assets, asset)
			continue
		}

		// Calculate value
		value := domainPortfolio.CalculateValue(tok.Decimal, balance, assetPrice)

		asset := &domainPortfolio.Asset{
			Token:  tok,
			Amount: balance,
			Price:  assetPrice,
			Value:  value,
			Source: "aggregated",
		}
		assets = append(assets, asset)

		s.logger.Debug("Created asset",
			zap.String("token", tok.Symbol),
			zap.String("amount", balance.String()),
			zap.String("value", value.String()))
	}

	s.logger.Info("Successfully created portfolio assets",
		zap.String("portfolio_id", portfolioID),
		zap.Int("asset_count", len(assets)))

	return portfolio, assets, nil
}
