package transaction

import (
	"testing"
)

func TestTransaction_ExtractMethodSignature(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid transfer signature",
			input:    "0xa9059cbb000000000000000000000000",
			expected: "0xa9059cbb",
		},
		{
			name:     "valid swap signature",
			input:    "0x7ff36ab5000000000000000000000000",
			expected: "0x7ff36ab5",
		},
		{
			name:     "short input",
			input:    "0x123",
			expected: "",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "uppercase hex",
			input:    "0xA9059CBB000000000000000000000000",
			expected: "0xa9059cbb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{}
			result := tx.ExtractMethodSignature(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractMethodSignature() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransaction_ClassifyType(t *testing.T) {
	tests := []struct {
		name      string
		methodSig string
		input     string
		expected  TransactionType
	}{
		{
			name:      "transfer method",
			methodSig: "0xa9059cbb",
			input:     "0xa9059cbb000000000000000000000000",
			expected:  TransactionTypeSend,
		},
		{
			name:      "transferFrom method",
			methodSig: "0x23b872dd",
			input:     "0x23b872dd000000000000000000000000",
			expected:  TransactionTypeSend,
		},
		{
			name:      "swap method (Uniswap V2)",
			methodSig: "0x7ff36ab5",
			input:     "0x7ff36ab5000000000000000000000000",
			expected:  TransactionTypeSwap,
		},
		{
			name:      "swap method (Uniswap V3)",
			methodSig: "0x02751cec",
			input:     "0x02751cec000000000000000000000000",
			expected:  TransactionTypeSwap,
		},
		{
			name:      "stake method",
			methodSig: "0x3d18b912",
			input:     "0x3d18b912000000000000000000000000",
			expected:  TransactionTypeStake,
		},
		{
			name:      "deposit method (staking)",
			methodSig: "0xb6b55f25",
			input:     "0xb6b55f25000000000000000000000000",
			expected:  TransactionTypeStake,
		},
		{
			name:      "empty input (ETH transfer)",
			methodSig: "",
			input:     "",
			expected:  TransactionTypeSend,
		},
		{
			name:      "unknown method defaults to send",
			methodSig: "0x12345678",
			input:     "0x12345678000000000000000000000000",
			expected:  TransactionTypeSend,
		},
		{
			name:      "extract from input when methodSig not set",
			methodSig: "",
			input:     "0xa9059cbb000000000000000000000000",
			expected:  TransactionTypeSend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{
				MethodSig: tt.methodSig,
			}
			result := tx.ClassifyType(tt.input)
			if result != tt.expected {
				t.Errorf("ClassifyType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransaction_DetectDirection(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		address  string
		expected TransactionDirection
	}{
		{
			name:     "outgoing transaction",
			from:     "0x123",
			to:       "0x456",
			address:  "0x123",
			expected: TransactionDirectionOut,
		},
		{
			name:     "incoming transaction",
			from:     "0x123",
			to:       "0x456",
			address:  "0x456",
			expected: TransactionDirectionIn,
		},
		{
			name:     "case insensitive - outgoing",
			from:     "0xABC",
			to:       "0xDEF",
			address:  "0xabc",
			expected: TransactionDirectionOut,
		},
		{
			name:     "case insensitive - incoming",
			from:     "0xABC",
			to:       "0xDEF",
			address:  "0xdef",
			expected: TransactionDirectionIn,
		},
		{
			name:     "default to out when neither matches",
			from:     "0x123",
			to:       "0x456",
			address:  "0x789",
			expected: TransactionDirectionOut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{
				From: tt.from,
				To:   tt.to,
			}
			result := tx.DetectDirection(tt.address)
			if result != tt.expected {
				t.Errorf("DetectDirection() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransaction_MethodName(t *testing.T) {
	tests := []struct {
		name      string
		methodSig string
		expected  string
	}{
		{
			name:      "transfer method",
			methodSig: "0xa9059cbb",
			expected:  "transfer",
		},
		{
			name:      "transferFrom method",
			methodSig: "0x23b872dd",
			expected:  "transferFrom",
		},
		{
			name:      "approve method",
			methodSig: "0x095ea7b3",
			expected:  "approve",
		},
		{
			name:      "swap method",
			methodSig: "0x7ff36ab5",
			expected:  "swapExactETHForTokens",
		},
		{
			name:      "stake method",
			methodSig: "0x3d18b912",
			expected:  "stake",
		},
		{
			name:      "deposit method",
			methodSig: "0xb6b55f25",
			expected:  "deposit",
		},
		{
			name:      "empty signature defaults to transfer",
			methodSig: "",
			expected:  "transfer",
		},
		{
			name:      "unknown signature",
			methodSig: "0x12345678",
			expected:  "unknown",
		},
		{
			name:      "uppercase signature",
			methodSig: "0xA9059CBB",
			expected:  "transfer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{
				MethodSig: tt.methodSig,
			}
			result := tx.MethodName()
			if result != tt.expected {
				t.Errorf("MethodName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransaction_CategorizationFlow(t *testing.T) {
	// Test complete categorization flow: extract signature -> classify type -> detect direction
	tx := &Transaction{
		From: "0x123",
		To:   "0x456",
	}

	// Extract method signature
	input := "0xa9059cbb000000000000000000000000"
	methodSig := tx.ExtractMethodSignature(input)
	if methodSig != "0xa9059cbb" {
		t.Errorf("ExtractMethodSignature() = %v, want 0xa9059cbb", methodSig)
	}

	// Set method signature and classify
	tx.MethodSig = methodSig
	txType := tx.ClassifyType(input)
	if txType != TransactionTypeSend {
		t.Errorf("ClassifyType() = %v, want %v", txType, TransactionTypeSend)
	}

	// Detect direction
	direction := tx.DetectDirection("0x123")
	if direction != TransactionDirectionOut {
		t.Errorf("DetectDirection() = %v, want %v", direction, TransactionDirectionOut)
	}

	// Verify method name
	methodName := tx.MethodName()
	if methodName != "transfer" {
		t.Errorf("MethodName() = %v, want transfer", methodName)
	}
}

func TestTransaction_AllSwapSignatures(t *testing.T) {
	// Test all known swap method signatures
	swapSignatures := []struct {
		sig      string
		name     string
		wantType TransactionType
	}{
		{"0x7ff36ab5", "swapExactETHForTokens", TransactionTypeSwap},
		{"0x02751cec", "swap", TransactionTypeSwap},
	}

	for _, tt := range swapSignatures {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{MethodSig: tt.sig}
			result := tx.ClassifyType("")
			if result != tt.wantType {
				t.Errorf("ClassifyType() for %s = %v, want %v", tt.name, result, tt.wantType)
			}
		})
	}
}

func TestTransaction_AllStakeSignatures(t *testing.T) {
	// Test all known staking method signatures
	stakeSignatures := []struct {
		sig      string
		name     string
		wantType TransactionType
	}{
		{"0x3d18b912", "stake", TransactionTypeStake},
		{"0xb6b55f25", "deposit", TransactionTypeStake},
	}

	for _, tt := range stakeSignatures {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{MethodSig: tt.sig}
			result := tx.ClassifyType("")
			if result != tt.wantType {
				t.Errorf("ClassifyType() for %s = %v, want %v", tt.name, result, tt.wantType)
			}
		})
	}
}
