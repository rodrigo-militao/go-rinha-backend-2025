package domain

import (
	"time"
)

//go:generate easyjson -all payment.go

//easyjson:json
type Payment struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
	Processor     string    `json:"processor,omitempty"`
}

//easyjson:json
type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
	RequestedAt   time.Time
}
