package application

import (
	"rinha-golang/internal/domain"
	"rinha-golang/internal/infra/database"
	"sync"
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

	var wg sync.WaitGroup
	wg.Add(2)

	// processa Default em paralelo
	go func() {
		count, total, _ := db.RangeQuerySummary(0, from, to)
		summary.Default.TotalRequests = count
		summary.Default.TotalAmount = float32(total) / 100
		wg.Done()
	}()

	// processa Fallback em paralelo
	go func() {
		count, total, _ := db.RangeQuerySummary(2, from, to)
		summary.Fallback.TotalRequests = count
		summary.Fallback.TotalAmount = float32(total) / 100
		wg.Done()
	}()

	wg.Wait()
	return
}
