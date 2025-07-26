package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"rinha-golang/internal/domain"
	"time"

	"github.com/google/uuid"
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

func (r *RedisPaymentRepository) AddToStream(ctx context.Context, payment domain.Payment) error {
	data, _ := json.Marshal(payment)
	return r.client.LPush(ctx, PAYMENTS_QUEUE, data).Err()
}

func (r *RedisPaymentRepository) StorePayment(ctx context.Context, payment domain.Payment) error {
	paymentData := map[string]interface{}{
		"correlationId": payment.CorrelationId.String(),
		"amount":        payment.Amount,
		"processor":     payment.Processor,
		"requestedAt":   payment.RequestedAt.Format(time.RFC3339Nano),
	}
	paymentJSON, err := json.Marshal(paymentData)
	if err != nil {
		return fmt.Errorf("failed to marshal payment data: %w", err)
	}
	return r.client.HSet(ctx, PAYMENTS_HASH, payment.CorrelationId.String(), paymentJSON).Err()
}

func (r *RedisPaymentRepository) GetAllPayments(ctx context.Context) ([]domain.Payment, error) {
	paymentsData, err := r.client.HGetAll(ctx, PAYMENTS_HASH).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payments: %w", err)
	}
	var payments []domain.Payment
	for _, paymentDataJSON := range paymentsData {
		var paymentData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentDataJSON), &paymentData); err != nil {
			continue
		}
		payment, err := parsePaymentFromData(paymentData)
		if err != nil {
			continue
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

func parsePaymentFromData(data map[string]interface{}) (domain.Payment, error) {
	correlationIdStr, ok := data["correlationId"].(string)
	if !ok {
		return domain.Payment{}, fmt.Errorf("invalid correlationId")
	}
	amount, ok := data["amount"].(float64)
	if !ok {
		return domain.Payment{}, fmt.Errorf("invalid amount")
	}
	processor, ok := data["processor"].(string)
	if !ok {
		return domain.Payment{}, fmt.Errorf("invalid processor")
	}
	requestedAtStr, ok := data["requestedAt"].(string)
	if !ok {
		return domain.Payment{}, fmt.Errorf("invalid requestedAt")
	}
	correlationId, err := uuid.Parse(correlationIdStr)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("failed to parse correlationId: %w", err)
	}
	requestedAt, err := time.Parse(time.RFC3339Nano, requestedAtStr)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("failed to parse requestedAt: %w", err)
	}
	return domain.Payment{
		CorrelationId: correlationId,
		Amount:        amount,
		RequestedAt:   requestedAt.UTC(),
		Processor:     processor,
	}, nil
}
