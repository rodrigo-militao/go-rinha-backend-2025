package redis

import (
	"context"
	"log"
	"rinha-golang/internal/domain"
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/redis/go-redis/v9"
)

var reqPayloadPool = sync.Pool{
	New: func() any {
		return new(ReqPayload)
	},
}

type Worker struct {
	Client      *redis.Client
	HostClients map[string]*fasthttp.HostClient
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

		payload := reqPayloadPool.Get().(*ReqPayload)
		*payload = ReqPayload{}

		if err := payload.UnmarshalJSON([]byte(result[1])); err != nil {
			log.Printf("[Worker %d] Failed to unmarshal payment: %v", w.WorkerNum, err)
			reqPayloadPool.Put(payload)
			continue
		}

		payment := domain.Payment{
			CorrelationId: payload.CorrelationId,
			Amount:        payload.Amount,
			RequestedAt:   time.Now().UTC(),
		}

		success := false
		maxRetries := 3

		for range maxRetries {
			if w.processPayment(ctx, payment, "default") {
				success = true
				break
			}
			time.Sleep(3 * time.Millisecond)
		}

		if !success {
			if w.processPayment(ctx, payment, "fallback") {
				success = true
			}
		}

		if !success {
			w.Client.RPush(ctx, PAYMENTS_QUEUE, result[1])
		}

		reqPayloadPool.Put(payload)
	}
}

func (w *Worker) processPayment(ctx context.Context, payment domain.Payment, processorName string) bool {
	client := w.HostClients[processorName]

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	uri := "http://" + client.Addr + "/payments"

	req.SetRequestURI(uri)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")

	payload := reqPayloadPool.Get().(*ReqPayload)
	*payload = ReqPayload{
		CorrelationId: payment.CorrelationId,
		Amount:        payment.Amount,
		RequestedAt:   payment.RequestedAt,
	}
	body, _ := payload.MarshalJSON()
	reqPayloadPool.Put(payload)
	req.SetBodyRaw(body)

	if err := client.Do(req, resp); err != nil {
		log.Printf("Error sending request: %v", err)
		return false
	}

	status := resp.StatusCode()
	if status < 200 || status >= 300 {
		return false
	}

	payment.Processor = processorName
	w.Repo.StorePayment(ctx, payment)
	return true
}
