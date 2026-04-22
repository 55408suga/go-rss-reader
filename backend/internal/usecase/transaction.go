package usecase

import "context"

// TransactionManager defines the interface for managing database transactions.
// Implementations must ensure that all operations within fn are executed atomically.
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
