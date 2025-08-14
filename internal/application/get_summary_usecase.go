package application

import (
	"rinha-golang/internal/domain"
	"rinha-golang/internal/infra/database"
	"time"
)

func GetSummary(
	db *database.MemDB,
	fromStr,
	toStr string,
) (summary domain.Summary, err error) {
	summary = domain.Summary{
		Default:  domain.SummaryItem{TotalRequests: 0, TotalAmount: 0},
		Fallback: domain.SummaryItem{TotalRequests: 0, TotalAmount: 0},
	}

	from := int64(0)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()

	if fromStr != "" {
		t, err := time.Parse(time.RFC3339Nano, fromStr)
		if err == nil {
			from = t.UnixNano()
		}
	}

	if toStr != "" {
		t, err := time.Parse(time.RFC3339Nano, toStr)
		if err == nil {
			to = t.UnixNano()
		}
	}

	data, err := db.RangeQuery(0, from, to)

	if err != nil {
		return
	}

	summary.Default.TotalRequests = len(data)
	var total int64
	for _, amount := range data {
		total += amount
	}
	summary.Default.TotalAmount = float32(total) / 100

	data, err = db.RangeQuery(2, from, to)

	if err != nil {
		return
	}

	var total1 int64
	for _, amount := range data {
		total1 += amount
	}
	summary.Fallback.TotalRequests = len(data)
	summary.Fallback.TotalAmount = float32(total1) / 100
	return

}
