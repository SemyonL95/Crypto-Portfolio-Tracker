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

type Repository interface {
	// GetTransactionsByAddress returns paginated transactions for an address
	GetTransactionsByAddress(ctx context.Context, address string, opts FilterOptions) (*TransactionResult, error)
	// GetAllTransactionsByAddress returns all transactions for an address (no pagination)
	GetAllTransactionsByAddress(ctx context.Context, address string, opts FilterOptions) ([]Transaction, error)
	GetTransactionByHash(ctx context.Context, hash string) (*Transaction, error)
	GetTokenBalances(ctx context.Context, address string) (map[string]string, error) // token address -> balance
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

	// Swap method signatures (DeFi)
	swapSig := "0x7ff36ab5"  // swapExactETHForTokens (Uniswap V2)
	swapSig2 := "0x02751cec" // swap (Uniswap V3)
	swapSig3 := "0x38ed1739" // swapExactTokensForTokens (Uniswap V2)
	swapSig4 := "0x8803dbee" // swapTokensForExactTokens (Uniswap V2)
	swapSig5 := "0x4a25d94a" // swapETHForExactTokens (Uniswap V2)
	swapSig6 := "0x791ac947" // swapExactTokensForETH (Uniswap V2)
	swapSig7 := "0x414bf389" // exactInputSingle (Uniswap V3)
	swapSig8 := "0xdb3e2198" // exactInput (Uniswap V3)
	swapSig9 := "0x5c11d795" // swapExactTokensForTokensSupportingFeeOnTransferTokens (Uniswap V2)

	// Staking method signatures
	stakeSig := "0x3d18b912"    // stake(uint256)
	depositSig := "0xb6b55f25"  // deposit(uint256) - common staking pattern
	stakeSig2 := "0x1249c58b"   // stake() - no parameters
	depositSig2 := "0xd0e30db0" // deposit() - payable
	depositSig3 := "0x47e7ef24" // deposit(uint256,address) - with recipient

	methodSig = strings.ToLower(methodSig)

	switch methodSig {
	case transferSig, transferFromSig:
		return TransactionTypeSend
	case swapSig, swapSig2, swapSig3, swapSig4, swapSig5, swapSig6, swapSig7, swapSig8, swapSig9:
		return TransactionTypeSwap
	case stakeSig, depositSig, stakeSig2, depositSig2, depositSig3:
		return TransactionTypeStake
	default:
		if input == "" || input == "0x" {
			return TransactionTypeSend
		}
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
		// ERC-20 methods
		"0xa9059cbb": "transfer",
		"0x23b872dd": "transferFrom",
		"0x095ea7b3": "approve",
		// Swap methods
		"0x7ff36ab5": "swapExactETHForTokens",
		"0x02751cec": "swap",
		"0x38ed1739": "swapExactTokensForTokens",
		"0x8803dbee": "swapTokensForExactTokens",
		"0x4a25d94a": "swapETHForExactTokens",
		"0x791ac947": "swapExactTokensForETH",
		"0x414bf389": "exactInputSingle",
		"0xdb3e2198": "exactInput",
		"0x5c11d795": "swapExactTokensForTokensSupportingFeeOnTransferTokens",
		// Staking methods
		"0x3d18b912": "stake",
		"0xb6b55f25": "deposit",
		"0x1249c58b": "stake",
		"0xd0e30db0": "deposit",
		"0x47e7ef24": "deposit",
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
