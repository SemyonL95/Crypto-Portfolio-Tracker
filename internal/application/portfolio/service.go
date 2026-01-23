package portfolio

import (
	"context"
	"errors"
	"math/big"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"time"
)

var (
	ErrInvalidHolding   = errors.New("invalid holding")
	ErrInvalidPortfolio = errors.New("invalid portfolio")
	ErrPortfolioExists  = errors.New("portfolio already exists")
)

type Service struct {
	repo          portfolio.PortfolioRepository
	priceProvider price.PriceProvider
}

func NewService(repo portfolio.PortfolioRepository, priceProvider price.PriceProvider) *Service {
	return &Service{
		repo:          repo,
		priceProvider: priceProvider,
	}
}

func (s *Service) CreatePortfolio(ctx context.Context, p *portfolio.Portfolio) error {
	if p == nil {
		return ErrInvalidPortfolio
	}

	if p.Address == "" {
		return ErrInvalidPortfolio
	}

	existingByAddr, err := s.repo.GetByAddress(ctx, p.Address)
	if err == nil && existingByAddr != nil {
		return portfolio.ErrPortfolioAddressExists
	}
	if err != nil && !errors.Is(err, portfolio.ErrPortfolioNotFound) {
		return err
	}

	if p.ID == "" {
		newPortfolio := portfolio.NewPortfolio("", p.Address)
		return s.repo.Save(ctx, newPortfolio)
	}

	existing, err := s.repo.Get(ctx, p.ID)
	if err == nil && existing != nil {
		return ErrPortfolioExists
	}

	if err != nil && !errors.Is(err, portfolio.ErrPortfolioNotFound) {
		return err
	}

	newPortfolio := portfolio.NewPortfolio(p.ID, p.Address)
	return s.repo.Save(ctx, newPortfolio)
}

func (s *Service) GetPortfolio(ctx context.Context, portfolioID string) (*portfolio.Portfolio, error) {
	p, err := s.repo.Get(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (s *Service) GetHoldings(ctx context.Context, portfolioID string) ([]*portfolio.Holding, error) {
	p, err := s.repo.Get(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	holdings := make([]*portfolio.Holding, len(p.Holdings))
	for i, h := range p.Holdings {
		holdings[i] = h
	}
	return holdings, nil
}

func (s *Service) AddHolding(ctx context.Context, portfolioID string, holding *portfolio.Holding) error {
	if holding.Token == nil {
		return ErrInvalidHolding
	}

	if holding.Amount == nil || holding.Amount.Sign() <= 0 {
		return ErrInvalidHolding
	}

	p, err := s.repo.Get(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, portfolio.ErrPortfolioNotFound) {
			return portfolio.ErrPortfolioNotFound
		}
		return err
	}

	existing := s.FindHoldingByToken(p, holding.Token.ID)
	if existing != nil {
		newAmount := new(big.Int).Add(existing.Amount, holding.Amount)
		s.UpdateAmount(existing, newAmount)

		return s.repo.UpdateHolding(ctx, portfolioID, existing)
	}

	// Create new holding with UUID and portfolioID
	newHolding := portfolio.NewHolding(portfolioID, "", holding.Token, holding.Amount)

	return s.repo.AddHolding(ctx, portfolioID, newHolding)
}

// UpdateHolding updates an existing holding
func (s *Service) UpdateHolding(ctx context.Context, portfolioID string, holdingID string, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return ErrInvalidHolding
	}

	p, err := s.repo.Get(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, portfolio.ErrPortfolioNotFound) {
			return portfolio.ErrPortfolioNotFound
		}
		return err
	}

	existing := s.FindHolding(p, holdingID)
	if existing == nil {
		return portfolio.ErrHoldingNotFound
	}

	s.UpdateAmount(existing, amount)

	return s.repo.UpdateHolding(ctx, portfolioID, existing)
}

// DeleteHolding removes a holding from the portfolio
func (s *Service) DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error {
	p, err := s.repo.Get(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, portfolio.ErrPortfolioNotFound) {
			return portfolio.ErrPortfolioNotFound
		}
		return err
	}

	holding := s.FindHolding(p, holdingID)
	if holding == nil {
		return portfolio.ErrHoldingNotFound
	}

	return s.repo.DeleteHolding(ctx, portfolioID, holdingID)
}

func (s *Service) UpdateAmount(holding *portfolio.Holding, amount *big.Int) {
	holding.Amount = new(big.Int).Set(amount)
	holding.UpdatedAt = time.Now()
}

func (s *Service) FindHolding(p *portfolio.Portfolio, holdingID string) *portfolio.Holding {
	for _, h := range p.Holdings {
		if h.ID == holdingID {
			return h
		}
	}
	return nil
}

func (s *Service) FindHoldingByToken(p *portfolio.Portfolio, tokenID string) *portfolio.Holding {
	for _, h := range p.Holdings {
		if h.Token != nil && h.Token.ID == tokenID {
			return h
		}
	}
	return nil
}

type PortfolioValue struct {
	Portfolio     *portfolio.Portfolio
	TotalValue    *big.Int
	Currency      string
	HoldingValues []*HoldingValue
	CalculatedAt  time.Time
}

type HoldingValue struct {
	Holding *portfolio.Holding
	Price   *price.Price
	Value   *big.Int
}
