package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"rinha-golang/internal/application"
	"rinha-golang/internal/config"
	"rinha-golang/internal/domain"
	"rinha-golang/internal/infra/database"
	"rinha-golang/internal/infra/worker"

	// _ "rinha-golang/internal/pprof"

	"github.com/valyala/fasthttp"
)

var pendingQueue chan []byte
var db = database.NewMemDB()
var cfg = config.Load()

var unixClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", cfg.OtherSocketPath)
		},
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	},
	Timeout: 3 * time.Second,
}

var BodyPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 100)
	},
}

func GetSummaryInternal(ctx *fasthttp.RequestCtx) {
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	summary, err := application.GetSummary(db, fromStr, toStr)

	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "failed to fetch data"}`)
		return
	}

	resp, err := json.Marshal(summary)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(resp)
}

func GetSummary(ctx *fasthttp.RequestCtx) {
	time.Sleep(50 * time.Millisecond)
	summaryOther := domain.Summary{
		Default:  domain.SummaryItem{TotalRequests: 0, TotalAmount: 0},
		Fallback: domain.SummaryItem{TotalRequests: 0, TotalAmount: 0},
	}
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	summary, err := application.GetSummary(db, fromStr, toStr)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "failed to fetch data"}`)
		return
	}

	req, err := http.NewRequest("GET", cfg.SummaryUrl, nil)
	values := req.URL.Query()
	values.Add("from", fromStr)
	values.Add("to", toStr)

	req.URL.RawQuery = values.Encode()

	res, err := unixClient.Do(req)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&summaryOther); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	summary.Default.TotalRequests += summaryOther.Default.TotalRequests
	summary.Default.TotalAmount += summaryOther.Default.TotalAmount

	summary.Fallback.TotalRequests += summaryOther.Fallback.TotalRequests
	summary.Fallback.TotalAmount += summaryOther.Fallback.TotalAmount

	resp, err := json.Marshal(summary)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(resp)
}

func handler(ctx *fasthttp.RequestCtx) {
	switch {
	case bytes.Equal(ctx.Path(), []byte("/payments")):
		ctx.SetStatusCode(fasthttp.StatusAccepted)
		buffer := BodyPool.Get().([]byte)[:0]
		buffer = append(buffer, ctx.PostBody()...)
		pendingQueue <- buffer
	case bytes.Equal(ctx.Path(), []byte("/payments-summary")):
		GetSummary(ctx)
	case bytes.Equal(ctx.Path(), []byte("/internal/payments-summary")):
		GetSummaryInternal(ctx)
	}
}

func main() {
	socketPath := filepath.Clean(cfg.SocketPath)
	if !filepath.IsAbs(socketPath) {
		socketPath = filepath.Join("/tmp", cfg.SocketPath)
	}

	// Remove socket antigo se existir
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	clients := map[string]*fasthttp.HostClient{
		"default": {
			Addr:                          "payment-processor-default:8080",
			MaxConns:                      1024,
			MaxIdleConnDuration:           30 * time.Second,
			ReadTimeout:                   3 * time.Second,
			WriteTimeout:                  3 * time.Second,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		},
		"fallback": {
			Addr:                          "payment-processor-fallback:8080",
			MaxConns:                      1024,
			MaxIdleConnDuration:           30 * time.Second,
			ReadTimeout:                   3 * time.Second,
			WriteTimeout:                  3 * time.Second,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		},
	}

	var paymentPool = sync.Pool{
		New: func() any {
			return new(domain.PaymentRequest)
		},
	}

	pendingQueue = make(chan []byte, 20_000)
	queue := make(chan *domain.PaymentRequest, 20_000)

	go worker.AddToQueue(pendingQueue, queue, &paymentPool, &BodyPool)
	go worker.WorkerPayments(db, clients, queue, &paymentPool)

	srv := &fasthttp.Server{
		Handler:                       handler,
		DisableHeaderNamesNormalizing: true,
		DisablePreParseMultipartForm:  true,
	}

	fmt.Printf("🔌 Servidor ouvindo no socket: %s\n", socketPath)
	err := srv.ListenAndServeUNIX(socketPath, 0777)
	if err != nil {
		panic(err)
	}

}
