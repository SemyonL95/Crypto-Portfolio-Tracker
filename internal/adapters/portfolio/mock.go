package portfolio

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testtask/internal/domain/portfolio"
	"time"
)

// Repository implements PortfolioRepository using in-memory storage
type Repository struct {
	mu              *sync.RWMutex
	portfolios      map[string]*portfolio.Portfolio          // key: portfolioID
	portfoliosByAddr map[string]*portfolio.Portfolio          // key: address -> portfolio (for uniqueness)
	holdings        map[string]map[string]*portfolio.Holding // key: portfolioID -> holdingID
}

// NewRepository creates a new in-memory portfolio repository
func NewRepository() *Repository {
	return &Repository{
		portfolios:       make(map[string]*portfolio.Portfolio),
		portfoliosByAddr: make(map[string]*portfolio.Portfolio),
		holdings:         make(map[string]map[string]*portfolio.Holding),
		mu:               &sync.RWMutex{},
	}
}

// Get retrieves a portfolio by portfolio ID
func (r *Repository) Get(ctx context.Context, portfolioID string) (*portfolio.Portfolio, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.portfolios[portfolioID]
	if !ok {
		return nil, fmt.Errorf("%w: portfolio_id=%s", portfolio.ErrPortfolioNotFound, portfolioID)
	}

	// Create a copy with holdings
	result := &portfolio.Portfolio{
		ID:        p.ID,
		Address:   p.Address,
		Holdings:  make([]*portfolio.Holding, 0),
		UpdatedAt: p.UpdatedAt,
	}

	// Load holdings for this portfolio
	if portfolioHoldings, ok := r.holdings[portfolioID]; ok {
		for _, h := range portfolioHoldings {
			// Create a copy of the holding
			holdingCopy := &portfolio.Holding{
				ID:          h.ID,
				PortfolioID: h.PortfolioID,
				Token:       h.Token,
				Amount:      new(big.Int).Set(h.Amount),
				CreatedAt:   h.CreatedAt,
				UpdatedAt:   h.UpdatedAt,
			}
			result.Holdings = append(result.Holdings, holdingCopy)
		}
	}

	return result, nil
}

// GetByAddress retrieves a portfolio by address
func (r *Repository) GetByAddress(ctx context.Context, address string) (*portfolio.Portfolio, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.portfoliosByAddr[address]
	if !ok {
		return nil, fmt.Errorf("%w: address=%s", portfolio.ErrPortfolioNotFound, address)
	}

	// Create a copy with holdings
	result := &portfolio.Portfolio{
		ID:        p.ID,
		Address:   p.Address,
		Holdings:  make([]*portfolio.Holding, 0),
		UpdatedAt: p.UpdatedAt,
	}

	// Load holdings for this portfolio
	if portfolioHoldings, ok := r.holdings[p.ID]; ok {
		for _, h := range portfolioHoldings {
			// Create a copy of the holding
			holdingCopy := &portfolio.Holding{
				ID:          h.ID,
				PortfolioID: h.PortfolioID,
				Token:       h.Token,
				Amount:      new(big.Int).Set(h.Amount),
				CreatedAt:   h.CreatedAt,
				UpdatedAt:   h.UpdatedAt,
			}
			result.Holdings = append(result.Holdings, holdingCopy)
		}
	}

	return result, nil
}

// Save saves or updates a portfolio
func (r *Repository) Save(ctx context.Context, p *portfolio.Portfolio) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if address is already used by a different portfolio
	if existing, ok := r.portfoliosByAddr[p.Address]; ok && existing.ID != p.ID {
		return fmt.Errorf("%w: address=%s", portfolio.ErrPortfolioAddressExists, p.Address)
	}

	// If updating an existing portfolio, remove old address mapping if address changed
	if existing, ok := r.portfolios[p.ID]; ok && existing.Address != p.Address {
		delete(r.portfoliosByAddr, existing.Address)
	}

	// Create a copy
	portfolioCopy := &portfolio.Portfolio{
		ID:        p.ID,
		Address:   p.Address,
		Holdings:  make([]*portfolio.Holding, 0),
		UpdatedAt: p.UpdatedAt,
	}

	r.portfolios[p.ID] = portfolioCopy
	r.portfoliosByAddr[p.Address] = portfolioCopy
	return nil
}

// GetHolding retrieves a holding by ID
func (r *Repository) GetHolding(ctx context.Context, portfolioID string, holdingID string) (*portfolio.Holding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	portfolioHoldings, ok := r.holdings[portfolioID]
	if !ok {
		return nil, fmt.Errorf("%w: holding_id=%s", portfolio.ErrHoldingNotFound, holdingID)
	}

	h, ok := portfolioHoldings[holdingID]
	if !ok {
		return nil, fmt.Errorf("%w: holding_id=%s", portfolio.ErrHoldingNotFound, holdingID)
	}

	// Return a copy
	return &portfolio.Holding{
		ID:          h.ID,
		PortfolioID: h.PortfolioID,
		Token:       h.Token,
		Amount:      new(big.Int).Set(h.Amount),
		CreatedAt:   h.CreatedAt,
		UpdatedAt:   h.UpdatedAt,
	}, nil
}

// AddHolding adds a holding to a portfolio
func (r *Repository) AddHolding(ctx context.Context, portfolioID string, holding *portfolio.Holding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify portfolio exists
	if _, ok := r.portfolios[portfolioID]; !ok {
		return fmt.Errorf("%w: portfolio_id=%s", portfolio.ErrPortfolioNotFound, portfolioID)
	}

	// Initialize holdings map for portfolio if needed
	if r.holdings[portfolioID] == nil {
		r.holdings[portfolioID] = make(map[string]*portfolio.Holding)
	}

	// Create a copy of the holding
	holdingCopy := &portfolio.Holding{
		ID:          holding.ID,
		PortfolioID: portfolioID,
		Token:       holding.Token,
		Amount:      new(big.Int).Set(holding.Amount),
		CreatedAt:   holding.CreatedAt,
		UpdatedAt:   holding.UpdatedAt,
	}

	r.holdings[portfolioID][holding.ID] = holdingCopy

	// Update portfolio timestamp
	if p, ok := r.portfolios[portfolioID]; ok {
		p.UpdatedAt = time.Now()
	}

	return nil
}

// UpdateHolding updates a holding
func (r *Repository) UpdateHolding(ctx context.Context, portfolioID string, holding *portfolio.Holding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	portfolioHoldings, ok := r.holdings[portfolioID]
	if !ok {
		return fmt.Errorf("%w: holding_id=%s", portfolio.ErrHoldingNotFound, holding.ID)
	}

	_, ok = portfolioHoldings[holding.ID]
	if !ok {
		return fmt.Errorf("%w: holding_id=%s", portfolio.ErrHoldingNotFound, holding.ID)
	}

	// Update the holding
	holdingCopy := &portfolio.Holding{
		ID:          holding.ID,
		PortfolioID: portfolioID,
		Token:       holding.Token,
		Amount:      new(big.Int).Set(holding.Amount),
		CreatedAt:   holding.CreatedAt,
		UpdatedAt:   holding.UpdatedAt,
	}

	r.holdings[portfolioID][holding.ID] = holdingCopy

	// Update portfolio timestamp
	if p, ok := r.portfolios[portfolioID]; ok {
		p.UpdatedAt = time.Now()
	}

	return nil
}

// DeleteHolding removes a holding from a portfolio
func (r *Repository) DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	portfolioHoldings, ok := r.holdings[portfolioID]
	if !ok {
		return fmt.Errorf("%w: holding_id=%s", portfolio.ErrHoldingNotFound, holdingID)
	}

	_, ok = portfolioHoldings[holdingID]
	if !ok {
		return fmt.Errorf("%w: holding_id=%s", portfolio.ErrHoldingNotFound, holdingID)
	}

	delete(portfolioHoldings, holdingID)

	// Update portfolio timestamp
	if p, ok := r.portfolios[portfolioID]; ok {
		p.UpdatedAt = time.Now()
	}

	return nil
}
