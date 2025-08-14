package database

import (
	"math"
	"rinha-golang/internal/domain"
	"sync"
)

type MemDB struct {
	mu   sync.RWMutex
	data map[int8][]domain.PaymentRequest
}

func NewMemDB() *MemDB {
	return &MemDB{
		data: make(map[int8][]domain.PaymentRequest),
	}
}

func (s *MemDB) Put(processor int8, payment domain.PaymentRequest) {
	s.data[processor] = append(s.data[processor], payment)
}

func (s *MemDB) RangeQuery(key int8, fromTs, toTs int64) ([]int64, error) {
	values := s.data[key]
	var amounts []int64

	for _, p := range values {
		timestamp := p.RequestedAt.UnixNano()

		if timestamp >= fromTs && timestamp <= toTs {
			amounts = append(amounts, int64(math.Round(float64(p.Amount*100))))

		} else if timestamp > toTs {
			break
		}
	}

	return amounts, nil
}
