package domain

type Summary struct {
	Default  SummaryItem `json:"default"`
	Fallback SummaryItem `json:"fallback"`
}

type SummaryItem struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}
