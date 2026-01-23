package transaction

import (
	"context"
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
	Amount       string
	Type         TransactionType
	Status       TransactionStatus
	Method       string
	MethodSig    string
	Direction    TransactionDirection
	Timestamp    time.Time
	BlockNumber  int64
}

type FilterOptions struct {
	Address string

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
	Transactions []Transaction
	Total        int
	Page         int
	PageSize     int
	TotalPages   int
}

type TransactionProvider interface {
	GetTransactions(ctx context.Context, opts FilterOptions) (*TransactionResult, error)
	GetTransactionByHash(ctx context.Context, hash string) (*Transaction, error)
	GetTransactionsByAddress(ctx context.Context, address string, opts FilterOptions) (*TransactionResult, error)
}

// ExtractMethodSignature extracts the method signature (first 4 bytes) from transaction input data
// Input format: "0x" + 4 bytes (8 hex chars) + data
// Returns the first 4 bytes (8 hex characters after "0x") in lowercase, or empty string if invalid
func (tx *Transaction) ExtractMethodSignature(input string) string {
	if len(input) < 10 {
		return ""
	}
	// Return first 4 bytes (8 hex characters after "0x")
	return strings.ToLower(input[:10])
}

// ClassifyType classifies transaction type based on method signature and input data
// It recognizes common ERC-20 methods, DeFi swap methods, and staking methods
func (tx *Transaction) ClassifyType(input string) TransactionType {
	// Use tx.MethodSig if available, otherwise extract from input
	methodSig := tx.MethodSig
	if methodSig == "" && input != "" {
		methodSig = tx.ExtractMethodSignature(input)
	}

	// Common ERC-20 method signatures
	transferSig := "0xa9059cbb"     // transfer(address,uint256)
	transferFromSig := "0x23b872dd" // transferFrom(address,address,uint256)
	// approveSig := "0x095ea7b3"   // approve(address,uint256) - not used in classification
	swapSig := "0x7ff36ab5"    // swapExactETHForTokens (Uniswap V2)
	swapSig2 := "0x02751cec"   // swap (Uniswap V3)
	stakeSig := "0x3d18b912"   // stake(uint256)
	depositSig := "0xb6b55f25" // deposit(uint256) - common staking pattern

	methodSig = strings.ToLower(methodSig)

	switch methodSig {
	case transferSig, transferFromSig:
		return TransactionTypeSend
	case swapSig, swapSig2:
		return TransactionTypeSwap
	case stakeSig, depositSig:
		return TransactionTypeStake
	default:
		// If it's a simple ETH transfer (no input or empty input)
		if input == "" || input == "0x" {
			return TransactionTypeSend
		}
		// Default to send for unknown methods
		return TransactionTypeSend
	}
}

func (tx *Transaction) DetectDirection(address string) TransactionDirection {
	from := strings.ToLower(tx.From)
	to := strings.ToLower(tx.To)
	address = strings.ToLower(address)

	if from == address {
		return TransactionDirectionOut
	}
	if to == address {
		return TransactionDirectionIn
	}
	// Default to out if neither matches (shouldn't happen in normal flow)
	return TransactionDirectionOut
}

func (tx *Transaction) MethodName() string {
	methodMap := map[string]string{
		"0xa9059cbb": "transfer",
		"0x23b872dd": "transferFrom",
		"0x095ea7b3": "approve",
		"0x7ff36ab5": "swapExactETHForTokens",
		"0x02751cec": "swap",
		"0x3d18b912": "stake",
		"0xb6b55f25": "deposit",
	}

	methodSig := strings.ToLower(tx.MethodSig)
	if name, ok := methodMap[methodSig]; ok {
		return name
	}

	if tx.MethodSig == "" || tx.MethodSig == "0x" {
		return "transfer"
	}

	return "unknown"
}
