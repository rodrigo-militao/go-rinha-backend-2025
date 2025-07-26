package domain

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	CorrelationId uuid.UUID
	Amount        float64
	RequestedAt   time.Time
	Processor     string // "default" or "fallback"
}
