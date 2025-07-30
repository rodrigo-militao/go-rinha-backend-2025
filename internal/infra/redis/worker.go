package redis

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"rinha-golang/internal/domain"
	"time"

	json "github.com/json-iterator/go"

	"github.com/redis/go-redis/v9"
)

type Worker struct {
	Client     *redis.Client
	HttpClient *http.Client
	Health     *HealthCheckService
	Repo       domain.PaymentRepository
	WorkerNum  int
}

func (w *Worker) Start(ctx context.Context) {
	processingQueue := fmt.Sprintf("payments:processing:%d", w.WorkerNum)
	for {
		result, err := w.Client.RPopLPush(ctx, PAYMENTS_QUEUE, processingQueue).Result()
		if err != nil {
			if err == redis.Nil {
				time.Sleep(1 * time.Second)
				continue
			}
			log.Printf("[Worker %d] Redis error: %v", w.WorkerNum, err)
			time.Sleep(1 * time.Second)
			continue
		}

		var reqPayload struct {
			CorrelationId string  `json:"correlationId"`
			Amount        float64 `json:"amount"`
		}
		if err := json.Unmarshal([]byte(result), &reqPayload); err != nil {
			log.Printf("[Worker %d] Failed to unmarshal payment: %v", w.WorkerNum, err)
			w.Client.LRem(ctx, processingQueue, 1, result)
			continue
		}

		payment := domain.Payment{
			CorrelationId: reqPayload.CorrelationId,
			Amount:        reqPayload.Amount,
			RequestedAt:   time.Now().UTC(),
		}

		processor := w.Health.GetCurrent()
		if !w.processPayment(ctx, payment, processor, w.HttpClient) {
			w.Client.LPush(ctx, PAYMENTS_QUEUE, result)
			w.Client.LRem(ctx, processingQueue, 1, result)
			continue
		}

		w.Client.LRem(ctx, processingQueue, 1, result)
	}
}

func (w *Worker) processPayment(ctx context.Context, payment domain.Payment, processor ProcessorStatus, client *http.Client) bool {
	if processor.URL == "" {
		log.Printf("[Worker %d] CRITICAL: Processor URL is empty!", w.WorkerNum)
		return false
	}

	b, _ := json.Marshal(payment)

	req, err := http.NewRequestWithContext(ctx, "POST", processor.URL, bytes.NewReader(b))
	if err != nil {
		log.Printf("[Worker %d] Failed to create request: %v", w.WorkerNum, err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		// log.Printf("[Worker %d] Processor %s HTTP error: %v", w.WorkerNum, processor.Service, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// log.Printf("[Worker %d] Processor %s returned status: %v", w.WorkerNum, processor.Service, resp.StatusCode)
		return false
	}

	payment.Processor = processor.Service
	if err := w.Repo.StorePayment(ctx, payment); err != nil {
		log.Printf("[Worker %d] CRITICAL: Payment accepted by processor but failed to save in Redis: %v", w.WorkerNum, err)
	}
	return true
}
