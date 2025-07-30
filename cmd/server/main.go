package main

import (
	"context"
	"log"
	"net/http"
	"time"

	// "net/http"
	"os"
	"os/signal"
	"rinha-golang/internal/application"
	"rinha-golang/internal/config"
	http_infra "rinha-golang/internal/infra/http"
	"sync"
	"syscall"

	redis_impl "rinha-golang/internal/infra/redis"

	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

func main() {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     30 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisURL,
		PoolSize:     200,
		MinIdleConns: 50,
	})

	paymentRepo := redis_impl.NewRedisPaymentRepository(redisClient)
	processPaymentUC := &application.ProcessPaymentUseCase{Repo: paymentRepo}
	getSummaryUC := &application.GetSummaryUseCase{Repo: paymentRepo}

	healthCheck := redis_impl.NewHealthCheckService(redisClient, cfg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		healthCheck.Start()
	}()

	log.Printf("Iniciando %d workers...", cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		worker := &redis_impl.Worker{
			Client:     redisClient,
			HttpClient: httpClient,
			Health:     healthCheck,
			Repo:       paymentRepo,
			WorkerNum:  i,
		}
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker.Start(ctx)
			log.Printf("Worker %d encerrado.", workerID)
		}(i + 1)
	}

	routes_handler := http_infra.SetupRoutes(processPaymentUC, getSummaryUC)

	server := &fasthttp.Server{
		Handler:     routes_handler,
		Name:        "rinha-go",
		Concurrency: 256 * 1024,
	}

	go func() {
		log.Printf("Servidor HTTP escutando em %s", cfg.ListenAddr)
		if err := server.ListenAndServe(cfg.ListenAddr); err != nil {
			log.Fatalf("Erro fatal no servidor HTTP: %v", err)
		}
	}()

	log.Println("Aplicação iniciada. Pressione Ctrl+C para encerrar.")
	<-ctx.Done()

	log.Println("Sinal de desligamento recebido...")
	if err := server.Shutdown(); err != nil {
		log.Printf("Erro no desligamento do servidor fasthttp: %v", err)
	}
	log.Println("Servidor HTTP encerrado.")

	log.Println("Aguardando todos os processos em background finalizarem...")

	wg.Wait()

	log.Println("Aplicação encerrada com sucesso.")
}
