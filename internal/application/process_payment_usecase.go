package application

import (
	"context"
	"log"
	"rinha-golang/internal/domain"
)

type ProcessPaymentUseCase struct {
	Repo domain.PaymentRepository
}

func (s *ProcessPaymentUseCase) Execute(ctx context.Context, payload []byte) error {
	err := s.Repo.AddToStream(ctx, payload)
	if err != nil {
		log.Printf("[Handler] Failed to enqueue payment: %v", err)
		return err
	}

	return nil
}

func (s *ProcessPaymentUseCase) PurgePayments(ctx context.Context) error {
	return s.Repo.PurgePayments(ctx)
}
