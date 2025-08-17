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
	s.mu.Lock()
	s.data[processor] = append(s.data[processor], payment)
	s.mu.Unlock()
}

// RangeQuerySummary evita criar slices grandes e já retorna os agregados
func (s *MemDB) RangeQuerySummary(key int8, fromTs, toTs int64) (count int, total int64, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := s.data[key]

	for _, p := range values {
		timestamp := p.RequestedAt.UnixNano()
		if timestamp >= fromTs && timestamp <= toTs {
			amount := int64(math.Round(float64(p.Amount * 100)))
			total += amount
			count++
		}
		// else if timestamp > toTs {
		// 	break
		// }
	}
	return count, total, nil
}
