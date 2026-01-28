package portfolio

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"testtask/internal/domain/holding"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/token"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepository{db: db}

	return repo, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, portfolioID string) (*portfolio.Portfolio, error) {
	query := `
		SELECT id, address, updated_at
		FROM portfolios
		WHERE id = ?
	`

	var p portfolio.Portfolio
	var updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, portfolioID).Scan(
		&p.ID,
		&p.Address,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: portfolio_id=%s", portfolio.ErrPortfolioNotFound, portfolioID)
		}
		return nil, fmt.Errorf("failed to get portfolio by ID: %w", err)
	}

	// Parse updated_at
	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		// Try parsing as datetime format if RFC3339 fails
		updatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}
	}
	p.UpdatedAt = updatedAt
	p.Holdings = make([]*holding.Holding, 0)

	return &p, nil
}

func (r *SQLiteRepository) GetByIDWithHoldings(ctx context.Context, portfolioID string) (*portfolio.Portfolio, error) {
	// First get the portfolio
	p, err := r.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// Load holdings for this portfolio
	holdings, err := r.loadHoldings(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to load holdings: %w", err)
	}

	// Assign holdings to portfolio
	p.Holdings = holdings

	return p, nil
}

func (r *SQLiteRepository) GetByAddress(ctx context.Context, address string) (*portfolio.Portfolio, error) {
	query := `
		SELECT id, address, updated_at
		FROM portfolios
		WHERE address = ?
	`

	var p portfolio.Portfolio
	var updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, address).Scan(
		&p.ID,
		&p.Address,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: address=%s", portfolio.ErrPortfolioNotFound, address)
		}
		return nil, fmt.Errorf("failed to get portfolio by address: %w", err)
	}

	// Parse updated_at
	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		// Try parsing as datetime format if RFC3339 fails
		updatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}
	}
	p.UpdatedAt = updatedAt
	p.Holdings = make([]*holding.Holding, 0)

	return &p, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, p *portfolio.Portfolio) error {
	// Check if address is already used by a different portfolio
	checkQuery := `SELECT id FROM portfolios WHERE address = ? AND id != ?`
	var existingID string
	err := r.db.QueryRowContext(ctx, checkQuery, p.Address, p.ID).Scan(&existingID)
	if err == nil {
		// Address exists and belongs to a different portfolio
		return fmt.Errorf("%w: address=%s", portfolio.ErrPortfolioAddressExists, p.Address)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing address: %w", err)
	}

	// Insert or update portfolio
	insertQuery := `
		INSERT INTO portfolios (id, address, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			address = excluded.address,
			updated_at = excluded.updated_at
	`

	updatedAtStr := p.UpdatedAt.Format(time.RFC3339)
	if p.UpdatedAt.IsZero() {
		updatedAtStr = time.Now().Format(time.RFC3339)
	}

	_, err = r.db.ExecContext(ctx, insertQuery, p.ID, p.Address, updatedAtStr)
	if err != nil {
		return fmt.Errorf("failed to create portfolio: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]*portfolio.Portfolio, error) {
	query := `
		SELECT id, address, updated_at
		FROM portfolios
		ORDER BY updated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list portfolios: %w", err)
	}
	defer rows.Close()

	var portfolios []*portfolio.Portfolio
	for rows.Next() {
		var p portfolio.Portfolio
		var updatedAtStr string

		if err := rows.Scan(&p.ID, &p.Address, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan portfolio: %w", err)
		}

		// Parse updated_at
		updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			// Try parsing as datetime format if RFC3339 fails
			updatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated_at: %w", err)
			}
		}
		p.UpdatedAt = updatedAt
		p.Holdings = make([]*holding.Holding, 0)

		portfolios = append(portfolios, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolios: %w", err)
	}

	return portfolios, nil
}

func (r *SQLiteRepository) ListWithHoldings(ctx context.Context) ([]*portfolio.Portfolio, error) {
	query := `
		SELECT 
			p.id, p.address, p.updated_at,
			h.id, h.portfolio_id, h.chain_id, h.token_id, h.token_symbol, h.token_address, 
			h.amount, h.created_at, h.updated_at
		FROM portfolios p
		LEFT JOIN holdings h ON p.id = h.portfolio_id
		ORDER BY p.updated_at DESC, h.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list portfolios with holdings: %w", err)
	}
	defer rows.Close()

	portfolioMap := make(map[string]*portfolio.Portfolio)
	var portfolios []*portfolio.Portfolio

	for rows.Next() {
		var portfolioID, address, portfolioUpdatedAtStr string
		var holdingID, holdingPortfolioID sql.NullString
		var chainID sql.NullInt64
		var tokenID, tokenSymbol, tokenAddress sql.NullString
		var amountStr, holdingCreatedAtStr, holdingUpdatedAtStr sql.NullString

		err := rows.Scan(
			&portfolioID,
			&address,
			&portfolioUpdatedAtStr,
			&holdingID,
			&holdingPortfolioID,
			&chainID,
			&tokenID,
			&tokenSymbol,
			&tokenAddress,
			&amountStr,
			&holdingCreatedAtStr,
			&holdingUpdatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Get or create portfolio
		p, exists := portfolioMap[portfolioID]
		if !exists {
			// Parse portfolio updated_at
			updatedAt, err := time.Parse(time.RFC3339, portfolioUpdatedAtStr)
			if err != nil {
				updatedAt, err = time.Parse("2006-01-02 15:04:05", portfolioUpdatedAtStr)
				if err != nil {
					return nil, fmt.Errorf("failed to parse portfolio updated_at: %w", err)
				}
			}

			p = &portfolio.Portfolio{
				ID:        portfolioID,
				Address:   address,
				UpdatedAt: updatedAt,
				Holdings:  make([]*holding.Holding, 0),
			}
			portfolioMap[portfolioID] = p
			portfolios = append(portfolios, p)
		}

		// If holding data exists, add it to the portfolio
		if holdingID.Valid {
			holding := holding.Holding{
				ID:          holdingID.String,
				PortfolioID: holdingPortfolioID.String,
				ChainID:     uint8(chainID.Int64),
			}

			// Reconstruct Token
			holding.Token = &token.Token{
				ID:      tokenID.String,
				Symbol:  tokenSymbol.String,
				Address: tokenAddress.String,
			}

			// Parse amount (big.Int from string)
			amount := new(big.Int)
			amount, ok := amount.SetString(amountStr.String, 10)
			if !ok {
				return nil, fmt.Errorf("failed to parse amount: %s", amountStr.String)
			}
			holding.Amount = amount

			// Parse timestamps
			createdAt, err := time.Parse(time.RFC3339, holdingCreatedAtStr.String)
			if err != nil {
				createdAt, err = time.Parse("2006-01-02 15:04:05", holdingCreatedAtStr.String)
				if err != nil {
					return nil, fmt.Errorf("failed to parse holding created_at: %w", err)
				}
			}
			holding.CreatedAt = createdAt

			updatedAt, err := time.Parse(time.RFC3339, holdingUpdatedAtStr.String)
			if err != nil {
				updatedAt, err = time.Parse("2006-01-02 15:04:05", holdingUpdatedAtStr.String)
				if err != nil {
					return nil, fmt.Errorf("failed to parse holding updated_at: %w", err)
				}
			}
			holding.UpdatedAt = updatedAt

			p.Holdings = append(p.Holdings, &holding)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return portfolios, nil
}

// loadHoldings loads all holdings for a given portfolio
func (r *SQLiteRepository) loadHoldings(ctx context.Context, portfolioID string) ([]*holding.Holding, error) {
	query := `
		SELECT id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at
		FROM holdings
		WHERE portfolio_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to query holdings: %w", err)
	}
	defer rows.Close()

	var holdings []*holding.Holding
	for rows.Next() {
		var h holding.Holding
		var tokenID, tokenSymbol, tokenAddress, amountStr, createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&h.ID,
			&h.PortfolioID,
			&h.ChainID,
			&tokenID,
			&tokenSymbol,
			&tokenAddress,
			&amountStr,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}

		// Reconstruct Token
		h.Token = &token.Token{
			ID:      tokenID,
			Symbol:  tokenSymbol,
			Address: tokenAddress,
		}

		// Parse amount (big.Int from string)
		amount := new(big.Int)
		amount, ok := amount.SetString(amountStr, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse amount: %s", amountStr)
		}
		h.Amount = amount

		// Parse timestamps
		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			createdAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
			}
		}
		h.CreatedAt = createdAt

		updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			updatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated_at: %w", err)
			}
		}
		h.UpdatedAt = updatedAt

		holdings = append(holdings, &h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating holdings: %w", err)
	}

	return holdings, nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// GetHolding retrieves a holding by portfolio ID and holding ID
func (r *SQLiteRepository) GetHolding(ctx context.Context, portfolioID string, holdingID string) (*holding.Holding, error) {
	query := `
		SELECT id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at
		FROM holdings
		WHERE portfolio_id = ? AND id = ?
	`

	var h holding.Holding
	var tokenID, tokenSymbol, tokenAddress, amountStr, createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, portfolioID, holdingID).Scan(
		&h.ID,
		&h.PortfolioID,
		&h.ChainID,
		&tokenID,
		&tokenSymbol,
		&tokenAddress,
		&amountStr,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: portfolio_id=%s, holding_id=%s", holding.ErrHoldingNotFound, portfolioID, holdingID)
		}
		return nil, fmt.Errorf("failed to get holding: %w", err)
	}

	// Reconstruct Token
	h.Token = &token.Token{
		ID:      tokenID,
		Symbol:  tokenSymbol,
		Address: tokenAddress,
	}

	// Parse amount
	amount := new(big.Int)
	amount, ok := amount.SetString(amountStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse amount: %s", amountStr)
	}
	h.Amount = amount

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		createdAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}
	}
	h.CreatedAt = createdAt

	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		updatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}
	}
	h.UpdatedAt = updatedAt

	return &h, nil
}

// CreateHolding creates a new holding
func (r *SQLiteRepository) CreateHolding(ctx context.Context, portfolioID string, h *holding.Holding) error {
	if h.Token == nil {
		return fmt.Errorf("token is required")
	}

	query := `
		INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	createdAtStr := h.CreatedAt.Format(time.RFC3339)
	if h.CreatedAt.IsZero() {
		createdAtStr = time.Now().Format(time.RFC3339)
	}

	updatedAtStr := h.UpdatedAt.Format(time.RFC3339)
	if h.UpdatedAt.IsZero() {
		updatedAtStr = time.Now().Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, query,
		h.ID,
		portfolioID,
		h.ChainID,
		h.Token.ID,
		h.Token.Symbol,
		h.Token.Address,
		h.Amount.String(),
		createdAtStr,
		updatedAtStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create holding: %w", err)
	}

	return nil
}

// UpdateHolding updates an existing holding
func (r *SQLiteRepository) UpdateHolding(ctx context.Context, portfolioID string, h *holding.Holding) error {
	if h.Token == nil {
		return fmt.Errorf("token is required")
	}

	query := `
		UPDATE holdings
		SET chain_id = ?, token_id = ?, token_symbol = ?, token_address = ?, amount = ?, updated_at = ?
		WHERE id = ? AND portfolio_id = ?
	`

	updatedAtStr := h.UpdatedAt.Format(time.RFC3339)
	if h.UpdatedAt.IsZero() {
		updatedAtStr = time.Now().Format(time.RFC3339)
	}

	result, err := r.db.ExecContext(ctx, query,
		h.ChainID,
		h.Token.ID,
		h.Token.Symbol,
		h.Token.Address,
		h.Amount.String(),
		updatedAtStr,
		h.ID,
		portfolioID,
	)
	if err != nil {
		return fmt.Errorf("failed to update holding: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: portfolio_id=%s, holding_id=%s", holding.ErrHoldingNotFound, portfolioID, h.ID)
	}

	return nil
}

// DeleteHolding deletes a holding
func (r *SQLiteRepository) DeleteHolding(ctx context.Context, portfolioID string, holdingID string) error {
	query := `DELETE FROM holdings WHERE id = ? AND portfolio_id = ?`

	result, err := r.db.ExecContext(ctx, query, holdingID, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to delete holding: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: portfolio_id=%s, holding_id=%s", holding.ErrHoldingNotFound, portfolioID, holdingID)
	}

	return nil
}

// ListByPortfolioID lists all holdings for a portfolio
func (r *SQLiteRepository) ListByPortfolioID(ctx context.Context, portfolioID string) ([]*holding.Holding, error) {
	return r.loadHoldings(ctx, portfolioID)
}
