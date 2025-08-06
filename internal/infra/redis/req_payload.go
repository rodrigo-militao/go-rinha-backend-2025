package redis

import "time"

//go:generate easyjson -all req_payload.go

//easyjson:json
type ReqPayload struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
}
