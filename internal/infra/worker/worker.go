package worker

import (
	"rinha-golang/internal/config"
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
	queue chan *domain.PaymentRequest,
	paymentPool *sync.Pool,
	cfg config.Config) {
	// httpClient := &http.Client{Timeout: 4 * time.Second}
	// httpClient := &fasthttp.Client{
	// 	ReadTimeout:                   5 * time.Second,
	// 	WriteTimeout:                  5 * time.Second,
	// 	MaxIdleConnDuration:           1 * time.Hour,
	// 	NoDefaultUserAgentHeader:      true, // Don't send: User-Agent: fasthttp
	// 	DisableHeaderNamesNormalizing: true, // If you set the case on your headers correctly you can enable this
	// 	DisablePathNormalizing:        true,
	// 	Dial: (&fasthttp.TCPDialer{
	// 		Concurrency:      4096,
	// 		DNSCacheDuration: time.Hour,
	// 	}).Dial,
	// }

	httpClient := map[string]*fasthttp.HostClient{
		"default": {
			Addr:                          "payment-processor-default:8080",
			MaxConns:                      4096,
			MaxIdleConnDuration:           1 * time.Hour,
			ReadTimeout:                   3 * time.Second,
			WriteTimeout:                  3 * time.Second,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			NoDefaultUserAgentHeader:      true,
			Dial: (&fasthttp.TCPDialer{
				Concurrency:      4096,
				DNSCacheDuration: time.Hour,
			}).Dial,
		},
		"fallback": {
			Addr:                          "payment-processor-fallback:8080",
			MaxConns:                      4096,
			MaxIdleConnDuration:           1 * time.Hour,
			ReadTimeout:                   3 * time.Second,
			WriteTimeout:                  3 * time.Second,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			NoDefaultUserAgentHeader:      true,
			Dial: (&fasthttp.TCPDialer{
				Concurrency:      4096,
				DNSCacheDuration: time.Hour,
			}).Dial,
		},
	}

	for {
		payment := <-queue
		processor := processPayment(httpClient, payment, cfg)
		if processor == -1 {
			queue <- payment
			continue
		}
		db.Put(processor, *payment)
		paymentPool.Put(payment)
	}
}

func processPayment(
	// client *http.Client,
	// client *fasthttp.Client,
	clients map[string]*fasthttp.HostClient,
	p *domain.PaymentRequest,
	cfg config.Config,
) int8 {
	maxRetries := 3

	for range maxRetries {
		defaultSuccess := gateway.PostPayment(clients["default"], p)
		if defaultSuccess {
			return int8(0)
		}
		time.Sleep(3 * time.Millisecond)
	}
	// defaultSuccess := gateway.PostPayment(client, p, cfg.ProcessorDefaultURL)

	// fallbackSuccess := gateway.PostPayment(client, p, cfg.ProcessorFallbackURL)
	fallbackSuccess := gateway.PostPayment(clients["fallback"], p)
	if fallbackSuccess {
		return int8(1)
	}

	return int8(-1)
}
