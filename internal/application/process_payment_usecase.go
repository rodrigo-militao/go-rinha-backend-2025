package application

import (
	"context"
	"rinha-golang/internal/domain"
)

type ProcessPaymentUseCase struct {
	Repo domain.PaymentRepository
}

func (s *ProcessPaymentUseCase) Execute(ctx context.Context, payload []byte) {
	s.Repo.AddToStream(ctx, payload)
}

func (s *ProcessPaymentUseCase) PurgePayments(ctx context.Context) error {
	return s.Repo.PurgePayments(ctx)
}
