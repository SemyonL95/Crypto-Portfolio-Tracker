package portfolio

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/token"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with schema for testing
func setupTestDB(t *testing.T) (*SQLiteRepository, func()) {
	// Use in-memory database for tests
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS portfolios (
		id TEXT PRIMARY KEY,
		address TEXT UNIQUE NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS holdings (
		id TEXT PRIMARY KEY,
		portfolio_id TEXT NOT NULL,
		chain_id INTEGER NOT NULL,
		token_id TEXT NOT NULL,
		token_symbol TEXT NOT NULL,
		token_address TEXT NOT NULL,
		amount TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_portfolios_address ON portfolios(address);
	CREATE INDEX IF NOT EXISTS idx_holdings_portfolio_id ON holdings(portfolio_id);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	repo := &SQLiteRepository{db: db}

	cleanup := func() {
		db.Close()
	}

	return repo, cleanup
}

// setupTestDBWithFile creates a temporary file-based SQLite database for testing
func setupTestDBWithFile(t *testing.T) (*SQLiteRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create schema manually since initSchema was removed
	schema := `
	CREATE TABLE IF NOT EXISTS portfolios (
		id TEXT PRIMARY KEY,
		address TEXT UNIQUE NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS holdings (
		id TEXT PRIMARY KEY,
		portfolio_id TEXT NOT NULL,
		chain_id INTEGER NOT NULL,
		token_id TEXT NOT NULL,
		token_symbol TEXT NOT NULL,
		token_address TEXT NOT NULL,
		amount TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_portfolios_address ON portfolios(address);
	CREATE INDEX IF NOT EXISTS idx_holdings_portfolio_id ON holdings(portfolio_id);
	`

	if _, err := repo.db.Exec(schema); err != nil {
		repo.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestSQLiteRepository_Create(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create new portfolio", func(t *testing.T) {
		p := portfolio.NewPortfolio("portfolio-1", "0x1234567890abcdef")
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v, want nil", err)
		}

		// Verify portfolio was created
		retrieved, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if retrieved.ID != p.ID {
			t.Errorf("GetByID() ID = %v, want %v", retrieved.ID, p.ID)
		}
		if retrieved.Address != p.Address {
			t.Errorf("GetByID() Address = %v, want %v", retrieved.Address, p.Address)
		}
	})

	t.Run("create portfolio with duplicate address", func(t *testing.T) {
		p1 := portfolio.NewPortfolio("portfolio-2", "0xabcdef1234567890")
		err := repo.Create(ctx, p1)
		if err != nil {
			t.Fatalf("Create() first portfolio error = %v", err)
		}

		p2 := portfolio.NewPortfolio("portfolio-3", "0xabcdef1234567890") // Same address
		err = repo.Create(ctx, p2)
		if err == nil {
			t.Error("Create() expected error for duplicate address, got nil")
		}

		if err != nil && err.Error() != "portfolio address already exists: address=0xabcdef1234567890" {
			t.Errorf("Create() error = %v, want address exists error", err)
		}
	})

	t.Run("update existing portfolio", func(t *testing.T) {
		p := portfolio.NewPortfolio("portfolio-4", "0x1111111111111111")
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Update the same portfolio
		p.Address = "0x2222222222222222"
		p.UpdatedAt = time.Now()
		err = repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() update error = %v", err)
		}

		// Verify update
		retrieved, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if retrieved.Address != "0x2222222222222222" {
			t.Errorf("GetByID() Address = %v, want 0x2222222222222222", retrieved.Address)
		}
	})
}

func TestSQLiteRepository_GetByID(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get existing portfolio", func(t *testing.T) {
		p := portfolio.NewPortfolio("portfolio-get-1", "0xget1234567890")
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		retrieved, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if retrieved.ID != p.ID {
			t.Errorf("GetByID() ID = %v, want %v", retrieved.ID, p.ID)
		}
		if retrieved.Address != p.Address {
			t.Errorf("GetByID() Address = %v, want %v", retrieved.Address, p.Address)
		}
		if len(retrieved.Holdings) != 0 {
			t.Errorf("GetByID() Holdings length = %v, want 0", len(retrieved.Holdings))
		}
	})

	t.Run("get non-existent portfolio", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "non-existent-id")
		if err == nil {
			t.Error("GetByID() expected error for non-existent portfolio, got nil")
		}

		if err != nil && err.Error() != "portfolio not found: portfolio_id=non-existent-id" {
			t.Errorf("GetByID() error = %v, want portfolio not found", err)
		}
	})
}

func TestSQLiteRepository_GetByAddress(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get existing portfolio by address", func(t *testing.T) {
		address := "0xaddress123456789"
		p := portfolio.NewPortfolio("portfolio-addr-1", address)
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		retrieved, err := repo.GetByAddress(ctx, address)
		if err != nil {
			t.Fatalf("GetByAddress() error = %v", err)
		}

		if retrieved.ID != p.ID {
			t.Errorf("GetByAddress() ID = %v, want %v", retrieved.ID, p.ID)
		}
		if retrieved.Address != address {
			t.Errorf("GetByAddress() Address = %v, want %v", retrieved.Address, address)
		}
	})

	t.Run("get non-existent portfolio by address", func(t *testing.T) {
		_, err := repo.GetByAddress(ctx, "0xnonexistent")
		if err == nil {
			t.Error("GetByAddress() expected error for non-existent address, got nil")
		}

		if err != nil && err.Error() != "portfolio not found: address=0xnonexistent" {
			t.Errorf("GetByAddress() error = %v, want portfolio not found", err)
		}
	})
}

func TestSQLiteRepository_List(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty portfolios", func(t *testing.T) {
		portfolios, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(portfolios) != 0 {
			t.Errorf("List() length = %v, want 0", len(portfolios))
		}
	})

	t.Run("list multiple portfolios", func(t *testing.T) {
		// Create multiple portfolios with different timestamps
		p1 := portfolio.NewPortfolio("list-1", "0xlist1111111111")
		time.Sleep(50 * time.Millisecond) // Ensure different timestamps
		p2 := portfolio.NewPortfolio("list-2", "0xlist2222222222")
		time.Sleep(50 * time.Millisecond)
		p3 := portfolio.NewPortfolio("list-3", "0xlist3333333333")

		// Update timestamps explicitly to ensure ordering
		p1.UpdatedAt = time.Now().Add(-2 * time.Second)
		p2.UpdatedAt = time.Now().Add(-1 * time.Second)
		p3.UpdatedAt = time.Now()

		err := repo.Create(ctx, p1)
		if err != nil {
			t.Fatalf("Create() p1 error = %v", err)
		}
		err = repo.Create(ctx, p2)
		if err != nil {
			t.Fatalf("Create() p2 error = %v", err)
		}
		err = repo.Create(ctx, p3)
		if err != nil {
			t.Fatalf("Create() p3 error = %v", err)
		}

		portfolios, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(portfolios) != 3 {
			t.Errorf("List() length = %v, want 3", len(portfolios))
		}

		// Verify order (should be DESC by updated_at, so p3 should be first)
		if portfolios[0].ID != p3.ID {
			t.Errorf("List() first portfolio ID = %v, want %v", portfolios[0].ID, p3.ID)
		}

		// Verify all portfolios are present
		portfolioIDs := make(map[string]bool)
		for _, p := range portfolios {
			portfolioIDs[p.ID] = true
			if len(p.Holdings) != 0 {
				t.Errorf("List() portfolio %s Holdings length = %v, want 0", p.ID, len(p.Holdings))
			}
		}

		if !portfolioIDs[p1.ID] || !portfolioIDs[p2.ID] || !portfolioIDs[p3.ID] {
			t.Error("List() missing expected portfolios")
		}
	})
}

func TestSQLiteRepository_ListWithHoldings(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list portfolios with holdings", func(t *testing.T) {
		// Create portfolios
		p1 := portfolio.NewPortfolio("holdings-1", "0xholdings111111")
		p2 := portfolio.NewPortfolio("holdings-2", "0xholdings222222")

		err := repo.Create(ctx, p1)
		if err != nil {
			t.Fatalf("Create() p1 error = %v", err)
		}
		err = repo.Create(ctx, p2)
		if err != nil {
			t.Fatalf("Create() p2 error = %v", err)
		}

		// Insert holdings directly into database
		now := time.Now()
		holdings := []struct {
			id           string
			portfolioID  string
			chainID      int
			tokenID      string
			tokenSymbol  string
			tokenAddress string
			amount       string
		}{
			{"holding-1", p1.ID, 1, "bitcoin", "BTC", "0xbtc", "100000000"},
			{"holding-2", p1.ID, 1, "ethereum", "ETH", "0xeth", "500000000000000000"},
			{"holding-3", p2.ID, 1, "bitcoin", "BTC", "0xbtc", "200000000"},
		}

		for _, h := range holdings {
			_, err := repo.db.Exec(`
				INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, h.id, h.portfolioID, h.chainID, h.tokenID, h.tokenSymbol, h.tokenAddress, h.amount, now.Format(time.RFC3339), now.Format(time.RFC3339))
			if err != nil {
				t.Fatalf("Failed to insert holding: %v", err)
			}
		}

		// List with holdings
		portfolios, err := repo.ListWithHoldings(ctx)
		if err != nil {
			t.Fatalf("ListWithHoldings() error = %v", err)
		}

		if len(portfolios) != 2 {
			t.Fatalf("ListWithHoldings() length = %v, want 2", len(portfolios))
		}

		// Find p1 and verify holdings
		var foundP1 *portfolio.Portfolio
		for _, p := range portfolios {
			if p.ID == p1.ID {
				foundP1 = p
				break
			}
		}

		if foundP1 == nil {
			t.Fatal("ListWithHoldings() p1 not found")
		}

		if len(foundP1.Holdings) != 2 {
			t.Errorf("ListWithHoldings() p1 Holdings length = %v, want 2", len(foundP1.Holdings))
		}

		// Verify holding details
		if foundP1.Holdings[0].Token.Symbol != "BTC" && foundP1.Holdings[0].Token.Symbol != "ETH" {
			t.Errorf("ListWithHoldings() holding token symbol = %v, want BTC or ETH", foundP1.Holdings[0].Token.Symbol)
		}
	})

	t.Run("list portfolios without holdings", func(t *testing.T) {
		p := portfolio.NewPortfolio("no-holdings-1", "0xnoholdings1111")
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		portfolios, err := repo.ListWithHoldings(ctx)
		if err != nil {
			t.Fatalf("ListWithHoldings() error = %v", err)
		}

		// Find the portfolio
		var found *portfolio.Portfolio
		for _, p := range portfolios {
			if p.ID == "no-holdings-1" {
				found = p
				break
			}
		}

		if found == nil {
			t.Fatal("ListWithHoldings() portfolio not found")
		}

		if len(found.Holdings) != 0 {
			t.Errorf("ListWithHoldings() Holdings length = %v, want 0", len(found.Holdings))
		}
	})

	t.Run("list multiple portfolios with mixed holdings", func(t *testing.T) {
		// Create portfolios
		p1 := portfolio.NewPortfolio("mixed-1", "0xmixed1111111")
		time.Sleep(10 * time.Millisecond)
		p2 := portfolio.NewPortfolio("mixed-2", "0xmixed2222222")

		err := repo.Create(ctx, p1)
		if err != nil {
			t.Fatalf("Create() p1 error = %v", err)
		}
		err = repo.Create(ctx, p2)
		if err != nil {
			t.Fatalf("Create() p2 error = %v", err)
		}

		// Insert holdings for p1 only
		now := time.Now()
		_, err = repo.db.Exec(`
			INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "mixed-holding-1", p1.ID, 1, "bitcoin", "BTC", "0xbtc", "100000000", now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("Failed to insert holding: %v", err)
		}

		portfolios, err := repo.ListWithHoldings(ctx)
		if err != nil {
			t.Fatalf("ListWithHoldings() error = %v", err)
		}

		// Verify p1 has holdings, p2 doesn't
		var foundP1, foundP2 *portfolio.Portfolio
		for _, p := range portfolios {
			if p.ID == p1.ID {
				foundP1 = p
			}
			if p.ID == p2.ID {
				foundP2 = p
			}
		}

		if foundP1 == nil || foundP2 == nil {
			t.Fatal("ListWithHoldings() portfolios not found")
		}

		if len(foundP1.Holdings) != 1 {
			t.Errorf("ListWithHoldings() p1 Holdings length = %v, want 1", len(foundP1.Holdings))
		}

		if len(foundP2.Holdings) != 0 {
			t.Errorf("ListWithHoldings() p2 Holdings length = %v, want 0", len(foundP2.Holdings))
		}
	})
}

func TestSQLiteRepository_HoldingsData(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("verify holdings data integrity", func(t *testing.T) {
		p := portfolio.NewPortfolio("data-test-1", "0xdatatest11111")
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Insert holding with large amount
		now := time.Now()
		largeAmount := "999999999999999999999999999"
		_, err = repo.db.Exec(`
			INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "data-holding-1", p.ID, 137, "polygon", "MATIC", "0xmatic", largeAmount, now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("Failed to insert holding: %v", err)
		}

		portfolios, err := repo.ListWithHoldings(ctx)
		if err != nil {
			t.Fatalf("ListWithHoldings() error = %v", err)
		}

		var found *portfolio.Portfolio
		for _, port := range portfolios {
			if port.ID == p.ID {
				found = port
				break
			}
		}

		if found == nil || len(found.Holdings) != 1 {
			t.Fatalf("ListWithHoldings() failed to retrieve holding")
		}

		h := found.Holdings[0]

		// Verify all fields
		if h.ID != "data-holding-1" {
			t.Errorf("Holding ID = %v, want data-holding-1", h.ID)
		}
		if h.PortfolioID != p.ID {
			t.Errorf("Holding PortfolioID = %v, want %v", h.PortfolioID, p.ID)
		}
		if h.ChainID != 137 {
			t.Errorf("Holding ChainID = %v, want 137", h.ChainID)
		}
		if h.Token.ID != "polygon" {
			t.Errorf("Holding Token.ID = %v, want polygon", h.Token.ID)
		}
		if h.Token.Symbol != "MATIC" {
			t.Errorf("Holding Token.Symbol = %v, want MATIC", h.Token.Symbol)
		}
		if h.Token.Address != "0xmatic" {
			t.Errorf("Holding Token.Address = %v, want 0xmatic", h.Token.Address)
		}

		// Verify big.Int amount
		expectedAmount := new(big.Int)
		expectedAmount.SetString(largeAmount, 10)
		if h.Amount.Cmp(expectedAmount) != 0 {
			t.Errorf("Holding Amount = %v, want %v", h.Amount.String(), expectedAmount.String())
		}
	})
}

func TestSQLiteRepository_FileBased(t *testing.T) {
	repo, cleanup := setupTestDBWithFile(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create and retrieve from file database", func(t *testing.T) {
		p := portfolio.NewPortfolio("file-test-1", "0xfiletest11111")
		err := repo.Create(ctx, p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		retrieved, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if retrieved.ID != p.ID {
			t.Errorf("GetByID() ID = %v, want %v", retrieved.ID, p.ID)
		}
	})
}

func TestSQLiteRepository_ConcurrentAccess(t *testing.T) {
	// Use file-based database for concurrent tests as in-memory has limitations
	repo, cleanup := setupTestDBWithFile(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("concurrent creates", func(t *testing.T) {
		// Create multiple portfolios concurrently
		done := make(chan bool, 10)
		errors := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				p := portfolio.NewPortfolio(fmt.Sprintf("concurrent-%d", id), fmt.Sprintf("0xconcurrent%d", id))
				err := repo.Create(ctx, p)
				if err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check for errors
		close(errors)
		for err := range errors {
			if err != nil {
				t.Errorf("Create() concurrent error = %v", err)
			}
		}

		portfolios, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		// Should have 10 portfolios (assuming no address conflicts)
		if len(portfolios) < 10 {
			t.Errorf("List() length = %v, want at least 10", len(portfolios))
		}
	})
}

func TestSQLiteRepository_ErrorHandling(t *testing.T) {
	t.Run("query on closed database", func(t *testing.T) {
		repo, cleanup := setupTestDB(t)
		cleanup() // Close the database

		ctx := context.Background()
		_, err := repo.GetByID(ctx, "test-id")
		if err == nil {
			t.Error("GetByID() expected error on closed database, got nil")
		}
	})

	t.Run("query with invalid SQL", func(t *testing.T) {
		repo, cleanup := setupTestDB(t)
		defer cleanup()

		ctx := context.Background()
		// Try to query a non-existent table
		_, err := repo.db.QueryContext(ctx, "SELECT * FROM non_existent_table")
		if err == nil {
			t.Error("QueryContext() expected error for invalid table, got nil")
		}
	})
}

// Helper function to create a test holding in the database
func insertTestHolding(t *testing.T, repo *SQLiteRepository, portfolioID, holdingID string, tok *token.Token, amount *big.Int) {
	now := time.Now()
	_, err := repo.db.Exec(`
		INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, holdingID, portfolioID, 1, tok.ID, tok.Symbol, tok.Address, amount.String(), now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to insert test holding: %v", err)
	}
}

func TestSQLiteRepository_Integration(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("full workflow", func(t *testing.T) {
		// 1. Create portfolios
		p1 := portfolio.NewPortfolio("workflow-1", "0xworkflow11111")
		p2 := portfolio.NewPortfolio("workflow-2", "0xworkflow22222")

		err := repo.Create(ctx, p1)
		if err != nil {
			t.Fatalf("Create() p1 error = %v", err)
		}
		err = repo.Create(ctx, p2)
		if err != nil {
			t.Fatalf("Create() p2 error = %v", err)
		}

		// 2. Insert holdings
		token1 := &token.Token{ID: "bitcoin", Symbol: "BTC", Address: "0xbtc"}
		token2 := &token.Token{ID: "ethereum", Symbol: "ETH", Address: "0xeth"}
		insertTestHolding(t, repo, p1.ID, "workflow-holding-1", token1, big.NewInt(100000000))
		insertTestHolding(t, repo, p1.ID, "workflow-holding-2", token2, big.NewInt(500000000000000000))
		insertTestHolding(t, repo, p2.ID, "workflow-holding-3", token1, big.NewInt(200000000))

		// 3. Get by ID (without holdings)
		retrieved, err := repo.GetByID(ctx, p1.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if len(retrieved.Holdings) != 0 {
			t.Errorf("GetByID() Holdings length = %v, want 0", len(retrieved.Holdings))
		}

		// 4. Get by address
		retrieved, err = repo.GetByAddress(ctx, p1.Address)
		if err != nil {
			t.Fatalf("GetByAddress() error = %v", err)
		}
		if retrieved.ID != p1.ID {
			t.Errorf("GetByAddress() ID = %v, want %v", retrieved.ID, p1.ID)
		}

		// 5. List (without holdings)
		portfolios, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(portfolios) != 2 {
			t.Errorf("List() length = %v, want 2", len(portfolios))
		}

		// 6. List with holdings
		portfolios, err = repo.ListWithHoldings(ctx)
		if err != nil {
			t.Fatalf("ListWithHoldings() error = %v", err)
		}

		// Find p1 and verify holdings
		var foundP1 *portfolio.Portfolio
		for _, p := range portfolios {
			if p.ID == p1.ID {
				foundP1 = p
				break
			}
		}

		if foundP1 == nil {
			t.Fatal("ListWithHoldings() p1 not found")
		}

		if len(foundP1.Holdings) != 2 {
			t.Errorf("ListWithHoldings() p1 Holdings length = %v, want 2", len(foundP1.Holdings))
		}

		// Verify holding data
		if foundP1.Holdings[0].Token.Symbol != "BTC" && foundP1.Holdings[0].Token.Symbol != "ETH" {
			t.Errorf("Holding Token.Symbol = %v, want BTC or ETH", foundP1.Holdings[0].Token.Symbol)
		}
	})
}
