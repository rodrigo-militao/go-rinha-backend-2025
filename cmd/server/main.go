package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
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

var unixClient = fasthttp.Client{
	Dial: func(addr string) (net.Conn, error) {
		return net.Dial("unix", cfg.OtherSocketPath)
	},
	ReadTimeout:                   3 * time.Second,
	WriteTimeout:                  3 * time.Second,
	MaxIdleConnDuration:           1 * time.Hour,
	DisableHeaderNamesNormalizing: true,
	DisablePathNormalizing:        true,
	NoDefaultUserAgentHeader:      true,
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

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	req.SetRequestURI(cfg.SummaryUrl)
	req.Header.SetMethod(fasthttp.MethodGet)

	req.URI().QueryArgs().Add("from", fromStr)
	req.URI().QueryArgs().Add("to", toStr)

	err = unixClient.Do(req, resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return
	}

	dec := json.NewDecoder(bytes.NewReader(resp.Body()))
	if err := dec.Decode(&summaryOther); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	summary.Default.TotalRequests += summaryOther.Default.TotalRequests
	summary.Default.TotalAmount += summaryOther.Default.TotalAmount

	summary.Fallback.TotalRequests += summaryOther.Fallback.TotalRequests
	summary.Fallback.TotalAmount += summaryOther.Fallback.TotalAmount

	res, err := json.Marshal(summary)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(res)
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
	case bytes.Equal(ctx.Path(), []byte("/health")):
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString("OK")
	}
}

func main() {
	socketPath := filepath.Clean(cfg.SocketPath)
	if !filepath.IsAbs(socketPath) {
		socketPath = filepath.Join("/tmp", cfg.SocketPath)
	}

	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	var paymentPool = sync.Pool{
		New: func() any {
			return new(domain.PaymentRequest)
		},
	}

	pendingQueue = make(chan []byte, 30_000)
	queue := make(chan *domain.PaymentRequest, 30_000)

	// go worker.AddToQueue(pendingQueue, queue, &paymentPool, &BodyPool)
	// go worker.WorkerPayments(db, queue, &paymentPool, cfg)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.AddToQueue(pendingQueue, queue, &paymentPool, &BodyPool)
	}()

	numWorkers, _ := strconv.Atoi(cfg.NumWorkers)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker.WorkerPayments(db, queue, &paymentPool, cfg)
		}(i + 1)
	}

	srv := &fasthttp.Server{
		Handler:                       handler,
		MaxConnsPerIP:                 0,
		DisableHeaderNamesNormalizing: true,
		DisablePreParseMultipartForm:  true,
	}

	fmt.Printf("🔌 Servidor ouvindo no socket: %s\n", socketPath)
	err := srv.ListenAndServeUNIX(socketPath, 0777)
	if err != nil {
		panic(err)
	}

	wg.Wait()

}
