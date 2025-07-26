package domain

import "context"

type PaymentRepository interface {
	AddToStream(ctx context.Context, payment Payment) error
	StorePayment(ctx context.Context, payment Payment) error
	GetAllPayments(ctx context.Context) ([]Payment, error)
	PurgePayments(ctx context.Context) error
}
