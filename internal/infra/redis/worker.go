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
		return new(domain.Payment)
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

		data := []byte(result[1])
		payment := reqPayloadPool.Get().(*domain.Payment)

		if err := payment.UnmarshalJSON(data); err != nil {
			reqPayloadPool.Put(payment)
			continue
		}

		success := false
		maxRetries := 3

		for range maxRetries {
			if w.processPayment(ctx, *payment, "default") {
				success = true
				break
			}
			time.Sleep(3 * time.Millisecond)
		}

		if !success {
			if w.processPayment(ctx, *payment, "fallback") {
				success = true
			}
		}

		reqPayloadPool.Put(payment)
	}
}

func (w *Worker) processPayment(ctx context.Context, payment domain.Payment, processorName string) bool {
	payment.Processor = processorName
	payment.RequestedAt = time.Now().UTC()
	client := w.HostClients[processorName]

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	uri := "http://" + client.Addr + "/payments"

	req.SetRequestURI(uri)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")

	body, _ := payment.MarshalJSON()
	req.SetBodyRaw(body)

	if err := client.Do(req, resp); err != nil {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return false
	}

	ok := resp.StatusCode() >= 200 && resp.StatusCode() < 300
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)

	if ok {
		w.Repo.StorePayment(ctx, payment.CorrelationId, body)
	}
	return ok
}
