package transaction

import (
	transactionadapter "testtask/internal/adapters/transaction"
	"testtask/internal/domain/transaction"
)

// NewTransactionProvider creates a configured TransactionProvider
// Currently uses Mock provider
func NewTransactionProvider() transaction.TransactionProvider {
	return transactionadapter.NewMockProvider()
}

