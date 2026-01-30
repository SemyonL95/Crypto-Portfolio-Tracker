package token

import (
	"context"
)

const ZeroAddress = "0x0000000000000000000000000000000000000"

type Token struct {
	ID      string
	Name    string
	Symbol  string
	Address string
	Decimal uint8
}

type Repository interface {
	GetList(ctx context.Context) ([]*Token, error)
	GetByAddress(ctx context.Context, address string) (*Token, error)
	GetByAddresses(ctx context.Context, addresses []string) map[string]*Token
}
