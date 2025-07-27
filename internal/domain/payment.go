package domain

import (
	"time"
)

type Payment struct {
	CorrelationId string
	Amount        float64
	RequestedAt   time.Time
	Processor     string // "default" or "fallback"
}
