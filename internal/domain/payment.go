package domain

import (
	"time"
)

type Payment struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
	Processor     string    `json:"processor,omitempty"`
}
