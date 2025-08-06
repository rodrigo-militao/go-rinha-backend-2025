package redis

import (
	"context"
	"fmt"
	"rinha-golang/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisPaymentRepository struct {
	client *redis.Client
}

func NewRedisPaymentRepository(client *redis.Client) *RedisPaymentRepository {
	return &RedisPaymentRepository{client: client}
}

func (r *RedisPaymentRepository) PurgePayments(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *RedisPaymentRepository) AddToStream(ctx context.Context, data []byte) {
	r.client.LPush(ctx, PAYMENTS_QUEUE, data)
}

func (r *RedisPaymentRepository) StorePayment(ctx context.Context, payment domain.Payment) {
	paymentJSON, err := payment.MarshalJSON()
	if err != nil {
		return
	}
	r.client.HSet(ctx, PAYMENTS_HASH, payment.CorrelationId, paymentJSON)
}

func (r *RedisPaymentRepository) GetAllPayments(ctx context.Context) ([]domain.Payment, error) {
	paymentsData, err := r.client.HGetAll(ctx, PAYMENTS_HASH).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payments: %w", err)
	}

	payments := make([]domain.Payment, 0, len(paymentsData))

	for _, paymentDataJSON := range paymentsData {
		var payment domain.Payment
		if err := payment.UnmarshalJSON([]byte(paymentDataJSON)); err == nil {
			payments = append(payments, payment)
		}
	}
	return payments, nil
}
