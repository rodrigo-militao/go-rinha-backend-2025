package redis

//go:generate easyjson

//easyjson:json
type HealthResponse struct {
	Failing bool `json:"failing"`
}
