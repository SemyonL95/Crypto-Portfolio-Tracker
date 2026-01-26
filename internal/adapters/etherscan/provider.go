package etherscan

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"testtask/internal/application/ratelimiter"
	"testtask/internal/domain/transaction"
)

// Provider implements transaction.Repository using Etherscan API
type Provider struct {
	client      *Client
	rateLimiter *ratelimiter.RateLimiter
}

// NewProvider creates a new Etherscan transaction provider
func NewProvider(client *Client, rateLimiter *ratelimiter.RateLimiter) *Provider {
	return &Provider{
		client:      client,
		rateLimiter: rateLimiter,
	}
}

// GetTransactionsByAddress retrieves transactions for a specific address with filtering and pagination
func (p *Provider) GetTransactionsByAddress(ctx context.Context, address string, opts transaction.FilterOptions) (*transaction.TransactionResult, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Apply rate limiting
	if err := p.rateLimiter.Allow(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	address = strings.ToLower(address)

	// Fetch all transaction types in parallel
	normalTxsChan := make(chan []transaction.Transaction, 1)
	tokenTxsChan := make(chan []transaction.Transaction, 1)
	errChan := make(chan error, 2)

	// Fetch normal ETH transactions
	go func() {
		defer close(normalTxsChan)
		txs, err := p.fetchNormalTransactions(ctx, address, opts)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch normal transactions: %w", err)
			return
		}
		normalTxsChan <- txs
	}()

	// Fetch ERC-20 token transfers
	go func() {
		defer close(tokenTxsChan)
		txs, err := p.fetchTokenTransfers(ctx, address, opts)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch token transfers: %w", err)
			return
		}
		tokenTxsChan <- txs
	}()

	// Get all filtered transactions
	filtered, err := p.getAllFilteredTransactions(ctx, address, opts, normalTxsChan, tokenTxsChan, errChan)
	if err != nil {
		return nil, err
	}

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

// GetAllTransactionsByAddress returns all transactions for an address without pagination
func (p *Provider) GetAllTransactionsByAddress(ctx context.Context, address string, opts transaction.FilterOptions) ([]transaction.Transaction, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Apply rate limiting
	if err := p.rateLimiter.Allow(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	address = strings.ToLower(address)

	// Fetch all transaction types in parallel
	normalTxsChan := make(chan []transaction.Transaction, 1)
	tokenTxsChan := make(chan []transaction.Transaction, 1)
	errChan := make(chan error, 2)

	// Fetch normal ETH transactions
	go func() {
		defer close(normalTxsChan)
		txs, err := p.fetchNormalTransactions(ctx, address, opts)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch normal transactions: %w", err)
			return
		}
		normalTxsChan <- txs
	}()

	// Fetch ERC-20 token transfers
	go func() {
		defer close(tokenTxsChan)
		txs, err := p.fetchTokenTransfers(ctx, address, opts)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch token transfers: %w", err)
			return
		}
		tokenTxsChan <- txs
	}()

	// Get all filtered transactions (no pagination)
	return p.getAllFilteredTransactions(ctx, address, opts, normalTxsChan, tokenTxsChan, errChan)
}

// getAllFilteredTransactions is a helper method that fetches, deduplicates, filters and sorts transactions
func (p *Provider) getAllFilteredTransactions(
	ctx context.Context,
	address string,
	opts transaction.FilterOptions,
	normalTxsChan <-chan []transaction.Transaction,
	tokenTxsChan <-chan []transaction.Transaction,
	errChan <-chan error,
) ([]transaction.Transaction, error) {
	// Collect results
	var allTxs []transaction.Transaction
	var errors []error

	for i := 0; i < 2; i++ {
		select {
		case txs := <-normalTxsChan:
			allTxs = append(allTxs, txs...)
		case txs := <-tokenTxsChan:
			allTxs = append(allTxs, txs...)
		case err := <-errChan:
			errors = append(errors, err)
		}
	}

	// If all requests failed, return error
	if len(errors) == 2 {
		return nil, fmt.Errorf("all API requests failed: %v", errors)
	}

	// Remove duplicates based on hash
	allTxs = p.deduplicateTransactions(allTxs)

	// Apply filters
	filtered := p.applyFilters(allTxs, opts, address)

	// Sort by timestamp (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	return filtered, nil
}

// GetTokenBalances calculates token balances by aggregating ERC-20 transfers
func (p *Provider) GetTokenBalances(ctx context.Context, address string) (map[string]string, error) {
	// Apply rate limiting
	if err := p.rateLimiter.Allow(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Fetch all token transfers for the address
	// We'll fetch a large number to get all transfers (Etherscan allows up to 10000)
	etherscanTxs, err := p.client.GetTokenTransfers(ctx, address, "", 0, 0, 1, 10000, "asc")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token transfers: %w", err)
	}

	// Aggregate balances by token address
	balances := make(map[string]*big.Int)
	addressLower := strings.ToLower(address)

	for _, etx := range etherscanTxs {
		tokenAddr := strings.ToLower(etx.ContractAddress)
		if tokenAddr == "" {
			continue
		}

		// Parse amount
		amount, ok := new(big.Int).SetString(etx.Value, 10)
		if !ok {
			continue
		}

		// Initialize balance if not exists
		if balances[tokenAddr] == nil {
			balances[tokenAddr] = big.NewInt(0)
		}

		// Add or subtract based on direction
		fromLower := strings.ToLower(etx.From)
		toLower := strings.ToLower(etx.To)

		if fromLower == addressLower {
			// Outgoing transfer - subtract
			balances[tokenAddr] = new(big.Int).Sub(balances[tokenAddr], amount)
		} else if toLower == addressLower {
			// Incoming transfer - add
			balances[tokenAddr] = new(big.Int).Add(balances[tokenAddr], amount)
		}
	}

	// Convert to string map
	result := make(map[string]string)
	for tokenAddr, balance := range balances {
		// Only include non-zero balances
		if balance.Sign() != 0 {
			result[tokenAddr] = balance.String()
		}
	}

	return result, nil
}

// fetchNormalTransactions fetches normal ETH transactions
func (p *Provider) fetchNormalTransactions(ctx context.Context, address string, opts transaction.FilterOptions) ([]transaction.Transaction, error) {
	// Calculate block range from date filters if provided
	startBlock := int64(0)
	endBlock := int64(0)
	if opts.FromDate != nil || opts.ToDate != nil {
		// For simplicity, we'll fetch all transactions and filter by date
		// In production, you might want to convert dates to block numbers
	}

	page := opts.Page
	if page < 1 {
		page = 1
	}
	offset := opts.PageSize
	if offset < 1 {
		offset = 20
	}

	etherscanTxs, err := p.client.GetNormalTransactions(ctx, address, startBlock, endBlock, page, offset, "desc")
	if err != nil {
		return nil, err
	}

	var txs []transaction.Transaction
	for _, etx := range etherscanTxs {
		tx := p.convertNormalTransaction(etx, address)
		txs = append(txs, tx)
	}

	return txs, nil
}

// fetchTokenTransfers fetches ERC-20 token transfers
func (p *Provider) fetchTokenTransfers(ctx context.Context, address string, opts transaction.FilterOptions) ([]transaction.Transaction, error) {
	contractAddress := ""
	if opts.Token != nil {
		contractAddress = *opts.Token
	}

	startBlock := int64(0)
	endBlock := int64(0)

	page := opts.Page
	if page < 1 {
		page = 1
	}
	offset := opts.PageSize
	if offset < 1 {
		offset = 20
	}

	etherscanTxs, err := p.client.GetTokenTransfers(ctx, address, contractAddress, startBlock, endBlock, page, offset, "desc")
	if err != nil {
		return nil, err
	}

	var txs []transaction.Transaction
	for _, etx := range etherscanTxs {
		tx := p.convertTokenTransfer(etx, address)
		txs = append(txs, tx)
	}

	return txs, nil
}

// fetchInternalTransactions fetches internal transactions
func (p *Provider) fetchInternalTransactions(ctx context.Context, address string, opts transaction.FilterOptions) ([]transaction.Transaction, error) {
	startBlock := int64(0)
	endBlock := int64(0)

	page := opts.Page
	if page < 1 {
		page = 1
	}
	offset := opts.PageSize
	if offset < 1 {
		offset = 20
	}

	etherscanTxs, err := p.client.GetInternalTransactions(ctx, address, startBlock, endBlock, page, offset, "desc")
	if err != nil {
		return nil, err
	}

	var txs []transaction.Transaction
	for _, etx := range etherscanTxs {
		tx := p.convertInternalTransaction(etx, address)
		txs = append(txs, tx)
	}

	return txs, nil
}

// convertNormalTransaction converts Etherscan normal transaction to domain transaction
func (p *Provider) convertNormalTransaction(etx EtherscanTransaction, address string) transaction.Transaction {
	timestamp, _ := ParseTimestamp(etx.TimeStamp)
	blockNumber, _ := parseBigInt(etx.BlockNumber)

	// Determine status
	status := transaction.TransactionStatusSuccess
	if etx.IsError == "1" || etx.TxReceiptStatus == "0" {
		status = transaction.TransactionStatusFailed
	}

	// Extract method signature
	methodSig := ""
	if len(etx.Input) >= 10 {
		methodSig = strings.ToLower(etx.Input[:10])
	}

	// Create transaction
	tx := transaction.Transaction{
		ID:           etx.Hash,
		Hash:         etx.Hash,
		From:         strings.ToLower(etx.From),
		To:           strings.ToLower(etx.To),
		TokenAddress: "",
		TokenSymbol:  "ETH",
		Amount:       etx.Value,
		Status:       status,
		MethodSig:    methodSig,
		Timestamp:    timestamp,
		BlockNumber:  blockNumber.Int64(),
	}

	// Classify type
	tx.Type = tx.ClassifyType(etx.Input)
	tx.Method = tx.MethodName()

	// Detect direction
	tx.Direction = tx.DetectDirection(address)

	// Adjust type based on direction for ETH transfers
	if tx.Type == transaction.TransactionTypeSend && tx.Direction == transaction.TransactionDirectionIn {
		tx.Type = transaction.TransactionTypeReceive
	}

	return tx
}

// convertTokenTransfer converts Etherscan token transfer to domain transaction
func (p *Provider) convertTokenTransfer(etx EtherscanTokenTransfer, address string) transaction.Transaction {
	timestamp, _ := ParseTimestamp(etx.TimeStamp)
	blockNumber, _ := parseBigInt(etx.BlockNumber)

	// Extract method signature
	methodSig := ""
	if len(etx.Input) >= 10 {
		methodSig = strings.ToLower(etx.Input[:10])
	}

	// Create transaction
	tx := transaction.Transaction{
		ID:           etx.Hash,
		Hash:         etx.Hash,
		From:         strings.ToLower(etx.From),
		To:           strings.ToLower(etx.To),
		TokenAddress: strings.ToLower(etx.ContractAddress),
		TokenSymbol:  etx.TokenSymbol,
		Amount:       etx.Value,
		Status:       transaction.TransactionStatusSuccess,
		MethodSig:    methodSig,
		Timestamp:    timestamp,
		BlockNumber:  blockNumber.Int64(),
	}

	// Classify type
	tx.Type = tx.ClassifyType(etx.Input)
	tx.Method = tx.MethodName()

	// Detect direction
	tx.Direction = tx.DetectDirection(address)

	// Adjust type based on direction
	if tx.Type == transaction.TransactionTypeSend && tx.Direction == transaction.TransactionDirectionIn {
		tx.Type = transaction.TransactionTypeReceive
	}

	return tx
}

// convertInternalTransaction converts Etherscan internal transaction to domain transaction
func (p *Provider) convertInternalTransaction(etx EtherscanInternalTransaction, address string) transaction.Transaction {
	timestamp, _ := ParseTimestamp(etx.TimeStamp)
	blockNumber, _ := parseBigInt(etx.BlockNumber)

	// Determine status
	status := transaction.TransactionStatusSuccess
	if etx.IsError == "1" {
		status = transaction.TransactionStatusFailed
	}

	// Extract method signature
	methodSig := ""
	if len(etx.Input) >= 10 {
		methodSig = strings.ToLower(etx.Input[:10])
	}

	// Create transaction
	tx := transaction.Transaction{
		ID:           etx.Hash,
		Hash:         etx.Hash,
		From:         strings.ToLower(etx.From),
		To:           strings.ToLower(etx.To),
		TokenAddress: strings.ToLower(etx.ContractAddress),
		TokenSymbol:  "ETH",
		Amount:       etx.Value,
		Status:       status,
		MethodSig:    methodSig,
		Timestamp:    timestamp,
		BlockNumber:  blockNumber.Int64(),
	}

	// Classify type
	tx.Type = tx.ClassifyType(etx.Input)
	tx.Method = tx.MethodName()

	// Detect direction
	tx.Direction = tx.DetectDirection(address)

	// Adjust type based on direction
	if tx.Type == transaction.TransactionTypeSend && tx.Direction == transaction.TransactionDirectionIn {
		tx.Type = transaction.TransactionTypeReceive
	}

	return tx
}

// applyFilters applies filter options to transactions
func (p *Provider) applyFilters(txs []transaction.Transaction, opts transaction.FilterOptions, address string) []transaction.Transaction {
	var filtered []transaction.Transaction

	for _, tx := range txs {
		// Type filter
		if opts.Type != nil && tx.Type != *opts.Type {
			continue
		}

		// Status filter
		if opts.Status != nil && tx.Status != *opts.Status {
			continue
		}

		// Token filter
		if opts.Token != nil {
			token := strings.ToLower(*opts.Token)
			if strings.ToLower(tx.TokenAddress) != token && strings.ToLower(tx.TokenSymbol) != token {
				continue
			}
		}

		// Date range filter
		if opts.FromDate != nil && tx.Timestamp.Before(*opts.FromDate) {
			continue
		}
		if opts.ToDate != nil && tx.Timestamp.After(*opts.ToDate) {
			continue
		}

		// Direction filter
		if opts.Direction != nil && tx.Direction != *opts.Direction {
			continue
		}

		filtered = append(filtered, tx)
	}

	return filtered
}

// deduplicateTransactions removes duplicate transactions based on hash
func (p *Provider) deduplicateTransactions(txs []transaction.Transaction) []transaction.Transaction {
	seen := make(map[string]bool)
	var unique []transaction.Transaction

	for _, tx := range txs {
		hash := strings.ToLower(tx.Hash)
		if !seen[hash] {
			seen[hash] = true
			unique = append(unique, tx)
		}
	}

	return unique
}

// GetTransactionByHash retrieves a specific transaction by its hash
func (p *Provider) GetTransactionByHash(ctx context.Context, hash string) (*transaction.Transaction, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	// Apply rate limiting
	if err := p.rateLimiter.Allow(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// For now, we'll search through transactions by fetching from an address
	// In a real implementation, you might want to use Etherscan's transaction status API
	// or maintain a local cache/index of transactions
	// This is a simplified implementation
	return nil, fmt.Errorf("GetTransactionByHash not fully implemented - requires transaction hash lookup")
}

// parseBigInt parses a string to big.Int
func parseBigInt(s string) (*big.Int, error) {
	bi := new(big.Int)
	_, ok := bi.SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse big int: %s", s)
	}
	return bi, nil
}
