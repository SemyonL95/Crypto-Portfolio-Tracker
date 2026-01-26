package chain

import "context"

type Chain struct {
	ID      string
	ChianID string
	Name    string
}

type Repository interface {
	GetList(ctx context.Context) ([]*Chain, error)
}
