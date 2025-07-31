package redis

import (
	"context"
	"fmt"
	"log"
	"rinha-golang/internal/domain"
	"time"

	json "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"

	"github.com/redis/go-redis/v9"
)

type Worker struct {
	Client      *redis.Client
	HostClients map[string]*fasthttp.HostClient
	Health      *HealthCheckService
	Repo        domain.PaymentRepository
	WorkerNum   int
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
		if !w.processPayment(ctx, payment, processor) {
			w.Client.LPush(ctx, PAYMENTS_QUEUE, result)
			w.Client.LRem(ctx, processingQueue, 1, result)
			continue
		}

		w.Client.LRem(ctx, processingQueue, 1, result)
	}
}

func (w *Worker) processPayment(ctx context.Context, payment domain.Payment, processor ProcessorStatus) bool {
	if processor.URL == "" {
		log.Printf("[Worker %d] CRITICAL: Processor URL is empty!", w.WorkerNum)
		return false
	}

	body, err := json.Marshal(payment)
	if err != nil {
		return false
	}

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(processor.URL)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetBody(body)

	client := w.HostClients[processor.Service]

	if err := client.Do(req, resp); err != nil {
		return false
	}

	status := resp.StatusCode()
	if status < 200 || status >= 300 {
		// log.Printf("[Worker %d] Erro na resposta: %d - %s", w.WorkerNum, status, resp.Body())
		return false
	}

	payment.Processor = processor.Service
	if err := w.Repo.StorePayment(ctx, payment); err != nil {
		log.Printf("[Worker %d] CRITICAL: Falha ao salvar pagamento: %v", w.WorkerNum, err)
	}
	return true
}
