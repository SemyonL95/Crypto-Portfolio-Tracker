package transaction

import (
	"context"
	"fmt"
	"strings"
	"time"

	"testtask/internal/domain/transaction"
)

// MockProvider implements TransactionProvider with mock data for testing
type MockProvider struct {
	transactions map[string]transaction.Transaction
	byAddress    map[string][]transaction.Transaction
}

// NewMockProvider creates a new mock transaction provider
func NewMockProvider() *MockProvider {
	now := time.Now()
	
	// Create sample transactions
	txs := []transaction.Transaction{
		{
			ID:           "0x1",
			Hash:         "0x1111111111111111111111111111111111111111111111111111111111111111",
			From:         "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:           "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			TokenAddress: "",
			TokenSymbol:  "ETH",
			Amount:       "1000000000000000000", // 1 ETH
			Type:         transaction.TransactionTypeSend,
			Status:       transaction.TransactionStatusSuccess,
			Method:       "transfer",
			MethodSig:    "0x",
			Direction:    transaction.TransactionDirectionOut,
			Timestamp:    now.Add(-2 * time.Hour),
			BlockNumber:  18000000,
		},
		{
			ID:           "0x2",
			Hash:         "0x2222222222222222222222222222222222222222222222222222222222222222",
			From:         "0xcccccccccccccccccccccccccccccccccccccccc",
			To:           "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			TokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
			TokenSymbol:  "USDT",
			Amount:       "500000000000000000000", // 500 USDT
			Type:         transaction.TransactionTypeReceive,
			Status:       transaction.TransactionStatusSuccess,
			Method:       "transfer",
			MethodSig:    "0xa9059cbb",
			Direction:    transaction.TransactionDirectionIn,
			Timestamp:    now.Add(-1 * time.Hour),
			BlockNumber:  18000001,
		},
		{
			ID:           "0x3",
			Hash:         "0x3333333333333333333333333333333333333333333333333333333333333333",
			From:         "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:           "0x7a250d5630b4cf539739df2c5dacb4c659f2488d", // Uniswap Router
			TokenAddress: "",
			TokenSymbol:  "ETH",
			Amount:       "500000000000000000", // 0.5 ETH
			Type:         transaction.TransactionTypeSwap,
			Status:       transaction.TransactionStatusSuccess,
			Method:       "swapExactETHForTokens",
			MethodSig:    "0x7ff36ab5",
			Direction:    transaction.TransactionDirectionOut,
			Timestamp:    now.Add(-30 * time.Minute),
			BlockNumber:  18000002,
		},
		{
			ID:           "0x4",
			Hash:         "0x4444444444444444444444444444444444444444444444444444444444444444",
			From:         "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:           "0xae78736cd615f374d3085123a210448e74fc6393", // rETH staking
			TokenAddress: "",
			TokenSymbol:  "ETH",
			Amount:       "2000000000000000000", // 2 ETH
			Type:         transaction.TransactionTypeStake,
			Status:       transaction.TransactionStatusSuccess,
			Method:       "stake",
			MethodSig:    "0x3d18b912",
			Direction:    transaction.TransactionDirectionOut,
			Timestamp:    now.Add(-15 * time.Minute),
			BlockNumber:  18000003,
		},
		{
			ID:           "0x5",
			Hash:         "0x5555555555555555555555555555555555555555555555555555555555555555",
			From:         "0xdddddddddddddddddddddddddddddddddddddddd",
			To:           "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			TokenAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", // USDC
			TokenSymbol:  "USDC",
			Amount:       "1000000000", // 1000 USDC (6 decimals)
			Type:         transaction.TransactionTypeReceive,
			Status:       transaction.TransactionStatusSuccess,
			Method:       "transfer",
			MethodSig:    "0xa9059cbb",
			Direction:    transaction.TransactionDirectionIn,
			Timestamp:    now.Add(-10 * time.Minute),
			BlockNumber:  18000004,
		},
		{
			ID:           "0x6",
			Hash:         "0x6666666666666666666666666666666666666666666666666666666666666666",
			From:         "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:           "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			TokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
			TokenSymbol:  "USDT",
			Amount:       "100000000000000000000", // 100 USDT
			Type:         transaction.TransactionTypeSend,
			Status:       transaction.TransactionStatusPending,
			Method:       "transfer",
			MethodSig:    "0xa9059cbb",
			Direction:    transaction.TransactionDirectionOut,
			Timestamp:    now.Add(-5 * time.Minute),
			BlockNumber:  18000005,
		},
		{
			ID:           "0x7",
			Hash:         "0x7777777777777777777777777777777777777777777777777777777777777777",
			From:         "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:           "0xffffffffffffffffffffffffffffffffffffffff",
			TokenAddress: "",
			TokenSymbol:  "ETH",
			Amount:       "100000000000000000", // 0.1 ETH
			Type:         transaction.TransactionTypeSend,
			Status:       transaction.TransactionStatusFailed,
			Method:       "transfer",
			MethodSig:    "0x",
			Direction:    transaction.TransactionDirectionOut,
			Timestamp:    now.Add(-1 * time.Minute),
			BlockNumber:  18000006,
		},
	}

	// Index transactions by hash and address
	byHash := make(map[string]transaction.Transaction)
	byAddr := make(map[string][]transaction.Transaction)

	for _, tx := range txs {
		byHash[tx.Hash] = tx
		// Index by both from and to addresses
		from := strings.ToLower(tx.From)
		to := strings.ToLower(tx.To)
		byAddr[from] = append(byAddr[from], tx)
		byAddr[to] = append(byAddr[to], tx)
	}

	return &MockProvider{
		transactions: byHash,
		byAddress:    byAddr,
	}
}

// GetTransactions retrieves transactions with filtering and pagination
func (p *MockProvider) GetTransactions(ctx context.Context, opts transaction.FilterOptions) (*transaction.TransactionResult, error) {
	if opts.Address == "" {
		return nil, fmt.Errorf("address is required for GetTransactions")
	}

	address := strings.ToLower(opts.Address)
	txs, exists := p.byAddress[address]
	if !exists {
		return &transaction.TransactionResult{
			Transactions: []transaction.Transaction{},
			Total:        0,
			Page:         opts.Page,
			PageSize:     opts.PageSize,
			TotalPages:   0,
		}, nil
	}

	// Apply filters
	filtered := make([]transaction.Transaction, 0)
	for _, tx := range txs {
		if p.matchesFilter(tx, opts) {
			filtered = append(filtered, tx)
		}
	}

	// Sort by timestamp (newest first)
	p.sortTransactionsByTimestamp(filtered)

	// Apply pagination
	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var paginatedTxs []transaction.Transaction
	if start < total {
		paginatedTxs = filtered[start:end]
	} else {
		paginatedTxs = []transaction.Transaction{}
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &transaction.TransactionResult{
		Transactions: paginatedTxs,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
	}, nil
}

// GetTransactionByHash retrieves a specific transaction by its hash
func (p *MockProvider) GetTransactionByHash(ctx context.Context, hash string) (*transaction.Transaction, error) {
	tx, exists := p.transactions[hash]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", hash)
	}
	return &tx, nil
}

// GetTransactionsByAddress retrieves transactions for a specific address
func (p *MockProvider) GetTransactionsByAddress(ctx context.Context, address string, opts transaction.FilterOptions) (*transaction.TransactionResult, error) {
	opts.Address = address
	return p.GetTransactions(ctx, opts)
}

// matchesFilter checks if a transaction matches the filter criteria
func (p *MockProvider) matchesFilter(tx transaction.Transaction, opts transaction.FilterOptions) bool {
	if opts.Type != nil && tx.Type != *opts.Type {
		return false
	}
	if opts.Status != nil && tx.Status != *opts.Status {
		return false
	}
	if opts.Token != nil {
		token := strings.ToLower(*opts.Token)
		if strings.ToLower(tx.TokenAddress) != token && strings.ToLower(tx.TokenSymbol) != token {
			return false
		}
	}
	if opts.FromDate != nil && tx.Timestamp.Before(*opts.FromDate) {
		return false
	}
	if opts.ToDate != nil && tx.Timestamp.After(*opts.ToDate) {
		return false
	}
	if opts.Direction != nil && tx.Direction != *opts.Direction {
		return false
	}
	return true
}

// sortTransactionsByTimestamp sorts transactions by timestamp (newest first)
func (p *MockProvider) sortTransactionsByTimestamp(txs []transaction.Transaction) {
	for i := 0; i < len(txs)-1; i++ {
		for j := i + 1; j < len(txs); j++ {
			if txs[i].Timestamp.Before(txs[j].Timestamp) {
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
	}
}

