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
	for {
		result, err := w.Client.BLPop(ctx, 0, PAYMENTS_QUEUE).Result()
		if err != nil {
			log.Printf("[Worker %d] Redis error: %v", w.WorkerNum, err)
			time.Sleep(1 * time.Second)
			continue
		}

		var reqPayload struct {
			CorrelationId string  `json:"correlationId"`
			Amount        float64 `json:"amount"`
		}
		if err := json.Unmarshal([]byte(result[1]), &reqPayload); err != nil {
			log.Printf("[Worker %d] Failed to unmarshal payment: %v", w.WorkerNum, err)
			continue
		}

		payment := domain.Payment{
			CorrelationId: reqPayload.CorrelationId,
			Amount:        reqPayload.Amount,
			RequestedAt:   time.Now().UTC(),
		}

		processor := w.Health.GetCurrent()
		if !w.processPayment(ctx, payment, processor) {
			w.Client.RPush(ctx, PAYMENTS_QUEUE, result[1])
			continue
		}
	}
}

func (w *Worker) processPayment(ctx context.Context, payment domain.Payment, processor ProcessorStatus) bool {
	if processor.URL == "" {
		log.Printf("[Worker %d] CRITICAL: Processor URL is empty!", w.WorkerNum)
		return false
	}

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(processor.URL)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")

	fmt.Fprintf(req.BodyWriter(), `{
        "correlationId": "%s",
        "amount": %f,
        "requestedAt": "%s"
    }`, payment.CorrelationId, payment.Amount, payment.RequestedAt.Format(time.RFC3339Nano))

	client := w.HostClients[processor.Service]

	if err := client.Do(req, resp); err != nil {
		return false
	}

	status := resp.StatusCode()
	if status < 200 || status >= 300 {
		return false
	}

	payment.Processor = processor.Service
	if err := w.Repo.StorePayment(ctx, payment); err != nil {
		log.Printf("[Worker %d] CRITICAL: Falha ao salvar pagamento: %v", w.WorkerNum, err)
	}
	return true
}
