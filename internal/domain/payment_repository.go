package domain

import "context"

type PaymentRepository interface {
	AddToStream(ctx context.Context, payload []byte)
	StorePayment(ctx context.Context, payment Payment)
	GetAllPayments(ctx context.Context) ([]Payment, error)
	PurgePayments(ctx context.Context) error
}
