package transaction

import (
	"context"
	"math/big"
	"sort"
	"strings"

	loggeradapter "testtask/internal/adapters/logger"
	"testtask/internal/domain/transaction"
)

// Service implements transaction aggregation and classification logic.
// It hides provider details (Etherscan, pagination, etc.) from callers.
type Service struct {
	provider transaction.Provider
	logger   *loggeradapter.Logger
}

func NewService(provider transaction.Provider, logger *loggeradapter.Logger) *Service {
	if logger == nil {
		logger = loggeradapter.NewNopLogger()
	}
	return &Service{
		provider: provider,
		logger:   logger,
	}
}

// GetTransactions fetches transactions for an address with filtering and pagination.
// Returns transactions, total count, and error.
func (s *Service) GetTransactions(
	ctx context.Context,
	address string,
	opts transaction.FilterOptions,
) ([]transaction.Transaction, int, error) {
	txns, err := s.TransactionsByAddress(ctx, address, opts)
	if err != nil {
		return nil, 0, err
	}

	// Convert to value slice for the response
	result := make([]transaction.Transaction, 0, len(txns))
	for _, tx := range txns {
		if tx != nil {
			result = append(result, *tx)
		}
	}

	return result, len(result), nil
}

// TransactionsByAddress aggregates normal, internal and token transactions
// for a given address, classifies direction and type, applies filtering
// and returns a (possibly paginated) slice.
func (s *Service) TransactionsByAddress(
	ctx context.Context,
	address string,
	opts transaction.FilterOptions,
) (transaction.Transactions, error) {
	addr := strings.ToLower(strings.TrimSpace(address))

	nativeTxs, err := s.provider.NativeTxsByAddress(ctx, addr, opts)
	if err != nil {
		return nil, err
	}
	internalTxs, err := s.provider.InternalTxsByAddress(ctx, addr, opts)
	if err != nil {
		return nil, err
	}
	tokenTxs, err := s.provider.TokenTxsByAddress(ctx, addr, opts)
	if err != nil {
		return nil, err
	}

	var all transaction.Transactions
	all = append(all, nativeTxs...)
	all = append(all, internalTxs...)
	all = append(all, tokenTxs...)

	// Enrich and filter.
	var filtered transaction.Transactions
	for _, tx := range all {
		s.enrichTransaction(tx, addr)
		if matchesFilter(tx, opts) {
			filtered = append(filtered, tx)
		}
	}

	// Sort by timestamp (newest first).
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply pagination on the filtered set.
	page := opts.Page
	pageSize := opts.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = len(filtered)
	}

	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return transaction.Transactions{}, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// enrichTransaction sets Direction and Type based on address and method data.
func (s *Service) enrichTransaction(tx *transaction.Transaction, address string) {
	if tx == nil {
		return
	}

	// Direction is already set by provider, but we can ensure it's correct
	tx.SetDirectionForAddress(address)

	// Default type based on direction if not already set.
	if tx.Type == "" {
		switch tx.Direction {
		case transaction.TransactionDirectionOut:
			tx.Type = transaction.TransactionTypeSend
		case transaction.TransactionDirectionIn:
			tx.Type = transaction.TransactionTypeReceive
		}
	}

	// Classify swap / stake using method signature or function name.
	methodSig := strings.ToLower(tx.MethodSig)
	methodName := strings.ToLower(tx.Method)

	if isSwap(methodSig, methodName) {
		tx.Type = transaction.TransactionTypeSwap
	} else if isStake(methodSig, methodName) {
		tx.Type = transaction.TransactionTypeStake
	}
}

func matchesFilter(tx *transaction.Transaction, opts transaction.FilterOptions) bool {
	if tx == nil {
		return false
	}

	if opts.Type != nil && tx.Type != *opts.Type {
		return false
	}

	if opts.Status != nil && tx.Status != *opts.Status {
		return false
	}

	if opts.Token != nil {
		token := strings.ToLower(strings.TrimSpace(*opts.Token))
		if token != "" && strings.ToLower(tx.TokenAddress) != token {
			return false
		}
	}

	if opts.Direction != nil && tx.Direction != *opts.Direction {
		return false
	}

	if opts.FromDate != nil && tx.Timestamp.Before(*opts.FromDate) {
		return false
	}

	if opts.ToDate != nil && tx.Timestamp.After(*opts.ToDate) {
		return false
	}

	return true
}

// isSwap detects common DEX swap functions by method signature or name.
func isSwap(methodSig, methodName string) bool {
	if methodSig == "" && methodName == "" {
		return false
	}

	swapSigs := map[string]struct{}{
		"0x38ed1739": {}, // swapExactTokensForTokens
		"0x18cbafe5": {}, // swapExactTokensForETH
		"0x7ff36ab5": {}, // swapExactETHForTokens
		"0x8803dbee": {}, // swapTokensForExactTokens
		"0xfb3bdb41": {}, // swapETHForExactTokens
	}

	if _, ok := swapSigs[methodSig]; ok {
		return true
	}

	if strings.Contains(methodName, "swap") {
		return true
	}

	return false
}

// isStake detects simple staking-related functions.
func isStake(methodSig, methodName string) bool {
	if methodSig == "" && methodName == "" {
		return false
	}

	if strings.Contains(methodName, "stake") ||
		strings.Contains(methodName, "deposit") ||
		strings.Contains(methodName, "enterstaking") {
		return true
	}

	return false
}

// CalculateBalanceFromHistory is a helper that computes native/token balances
// from the given transactions using integer arithmetic only.
func (s *Service) CalculateBalanceFromHistory(
	txns transaction.Transactions,
) (map[string]*big.Int, error) {
	return txns.CalculateTokensAmounts()
}
