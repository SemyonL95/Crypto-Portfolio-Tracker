package transaction

import (
	"context"
	"fmt"
	"testtask/internal/domain/transaction"
	httpports "testtask/internal/ports/http"
)

// Service implements TransactionService interface for HTTP layer
type Service struct {
	provider transaction.TransactionProvider
}

// NewService creates a new transaction service
func NewService(provider transaction.TransactionProvider) *Service {
	return &Service{
		provider: provider,
	}
}

// GetTransactions retrieves transactions with filtering and pagination
func (s *Service) GetTransactions(ctx context.Context, filters httpports.TransactionFilters) ([]transaction.Transaction, int, error) {
	// Convert HTTP filters to domain FilterOptions
	opts := transaction.FilterOptions{
		Page:     filters.Page,
		PageSize: filters.PageSize,
	}

	// Convert optional filters
	if filters.Type != nil {
		txType := transaction.TransactionType(*filters.Type)
		opts.Type = &txType
	}
	if filters.Status != nil {
		txStatus := transaction.TransactionStatus(*filters.Status)
		opts.Status = &txStatus
	}
	if filters.Token != nil {
		opts.Token = filters.Token
	}
	if filters.FromDate != nil {
		opts.FromDate = filters.FromDate
	}
	if filters.ToDate != nil {
		opts.ToDate = filters.ToDate
	}

	// Address is required for GetTransactions
	if filters.Address == nil || *filters.Address == "" {
		return nil, 0, fmt.Errorf("address is required for GetTransactions")
	}
	opts.Address = *filters.Address

	result, err := s.provider.GetTransactions(ctx, opts)
	if err != nil {
		return nil, 0, err
	}

	return result.Transactions, result.Total, nil
}

// GetTransactionByHash retrieves a specific transaction by its hash
func (s *Service) GetTransactionByHash(ctx context.Context, hash string) (*transaction.Transaction, error) {
	return s.provider.GetTransactionByHash(ctx, hash)
}

