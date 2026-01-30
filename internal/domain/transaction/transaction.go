package transaction

import (
	"context"
	"math/big"
	"strings"
	"time"
)

type TransactionType string

const (
	TransactionTypeSend    TransactionType = "send"
	TransactionTypeReceive TransactionType = "receive"
	TransactionTypeSwap    TransactionType = "swap"
	TransactionTypeStake   TransactionType = "stake"
)

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

type TransactionDirection string

const (
	TransactionDirectionIn  TransactionDirection = "in"
	TransactionDirectionOut TransactionDirection = "out"
)

type Transaction struct {
	ID           string
	Hash         string
	From         string
	To           string
	TokenAddress string
	TokenSymbol  string
	Amount       *big.Int // Changed from string to *big.Int
	Type         TransactionType
	Status       TransactionStatus
	GasPrice     *big.Int
	GasUsed      *big.Int
	Method       string
	MethodSig    string
	Direction    TransactionDirection
	Timestamp    time.Time
	BlockNumber  int64
}

// SetDirectionForAddress sets the Direction field based on from/to address comparison.
// This is used to determine if a transaction is incoming or outgoing for a specific address.
func (tx *Transaction) SetDirectionForAddress(address string) {
	if tx == nil {
		return
	}
	addr := strings.ToLower(address)
	from := strings.ToLower(tx.From)
	to := strings.ToLower(tx.To)

	switch {
	case from == addr && to != addr:
		tx.Direction = TransactionDirectionOut
	case to == addr && from != addr:
		tx.Direction = TransactionDirectionIn
	}
}

type FilterOptions struct {
	Address   string // Address to filter by (used for direction calculation)
	Type      *TransactionType
	Status    *TransactionStatus
	Token     *string // Token address or symbol
	FromDate  *time.Time
	ToDate    *time.Time
	Direction *TransactionDirection

	Page     int
	PageSize int
}

type TransactionResult struct {
	Transactions Transactions
	Total        int
	Page         int
	PageSize     int
	TotalPages   int
}

type Provider interface {
	NativeTxsByAddress(ctx context.Context, address string, opts FilterOptions) ([]*Transaction, error)
	TokenTxsByAddress(ctx context.Context, address string, opts FilterOptions) ([]*Transaction, error)
	InternalTxsByAddress(ctx context.Context, address string, opts FilterOptions) ([]*Transaction, error)
	GetNativeBalance(ctx context.Context, address string) (*big.Int, error)
}

type AggregatedData struct {
	Address            string
	Transactions       []Transaction
	ETHBalance         *big.Int
	CalculatedBalances map[string]*big.Int
}

type Transactions []*Transaction

// CalculateTokensAmounts calculates token balances derived only from
// transaction history. The map key is the token contract address
// (empty string for native token). Positive values mean net incoming
// funds, negative values mean net outgoing funds.
// Uses integer arithmetic only (big.Int).
func (ts *Transactions) CalculateTokensAmounts() (map[string]*big.Int, error) {
	if ts == nil {
		return map[string]*big.Int{}, nil
	}

	balances := make(map[string]*big.Int)

	for _, tx := range *ts {
		if tx == nil || tx.Amount == nil {
			continue
		}

		// Skip transactions with unknown direction to avoid corrupting balances
		if tx.Direction == "" {
			continue
		}

		tokenKey := strings.ToLower(tx.TokenAddress)

		if _, exists := balances[tokenKey]; !exists {
			balances[tokenKey] = big.NewInt(0)
		}

		switch tx.Direction {
		case TransactionDirectionIn:
			balances[tokenKey].Add(balances[tokenKey], tx.Amount)
		case TransactionDirectionOut:
			balances[tokenKey].Sub(balances[tokenKey], tx.Amount)
		}
	}

	return balances, nil
}
