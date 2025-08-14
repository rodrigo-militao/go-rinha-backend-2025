package worker

import (
	"rinha-golang/internal/domain"
	"rinha-golang/internal/infra/database"
	"rinha-golang/internal/infra/gateway"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

func AddToQueue(
	pendingQueue chan []byte,
	queue chan *domain.PaymentRequest,
	paymentPool *sync.Pool,
	BodyPool *sync.Pool,
) {
	for {
		body := <-pendingQueue
		p := paymentPool.Get().(*domain.PaymentRequest)

		p.CorrelationId = ""
		p.Amount = 0

		p.UnmarshalJSON(body)

		p.RequestedAt = time.Now().UTC()
		queue <- p
		BodyPool.Put(body)
	}
}

func WorkerPayments(
	db *database.MemDB,
	hostClients map[string]*fasthttp.HostClient,
	queue chan *domain.PaymentRequest,
	paymentPool *sync.Pool) {

	for {
		payment := <-queue
		processor := processPayment(hostClients, payment)
		if processor == -1 {
			// queue <- payment
			continue
		}
		db.Put(processor, *payment)
		paymentPool.Put(payment)
	}
}

func processPayment(
	hostClients map[string]*fasthttp.HostClient,
	p *domain.PaymentRequest,
) int8 {
	defaultSuccess := gateway.PostPayment(hostClients["default"], p)
	if defaultSuccess {
		return int8(0)
	}

	fallbackSuccess := gateway.PostPayment(hostClients["fallback"], p)
	if fallbackSuccess {
		return int8(1)
	}

	return int8(-1)
}
