package main

import (
	"context"
	"log"
	"time"

	"os"
	"os/signal"
	"rinha-golang/internal/application"
	"rinha-golang/internal/config"
	http_infra "rinha-golang/internal/infra/http"
	"sync"
	"syscall"

	redis_impl "rinha-golang/internal/infra/redis"

	// _ "rinha-golang/internal/pprof"

	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	workerCount := cfg.Workers

	// redisClient := redis.NewClient(&redis.Options{
	// 	Addr:         cfg.RedisURL,
	// 	PoolSize:     200,
	// 	MinIdleConns: 100,
	// })

	redisClient := redis.NewClient(&redis.Options{
		Network: "unix",
		Addr:    "/tmp/redis.sock",
	})

	if workerCount > 0 {
		startWorkers(ctx, redisClient, cfg, workerCount)
	} else {
		startAPI(ctx, redisClient, cfg)
	}

	warmUp(redisClient)
}

func startAPI(ctx context.Context, redisClient *redis.Client, cfg config.Config) {
	repo := redis_impl.NewRedisPaymentRepository(redisClient)

	processPaymentUC := &application.ProcessPaymentUseCase{
		Repo: repo,
	}

	getSummaryUC := &application.GetSummaryUseCase{
		Repo: repo,
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

	log.Println("API iniciada. Pressione Ctrl+C para encerrar.")
	<-ctx.Done()

	log.Println("Encerrando servidor HTTP...")
	if err := server.Shutdown(); err != nil {
		log.Printf("Erro ao encerrar HTTP: %v", err)
	}
	log.Println("Servidor HTTP encerrado.")
}

func startWorkers(ctx context.Context, redisClient *redis.Client, cfg config.Config, workerCount int) {
	log.Printf("Iniciando %d workers...", workerCount)

	clients := map[string]*fasthttp.HostClient{
		"default": {
			Addr:                "payment-processor-default:8080",
			MaxConns:            1024,
			MaxIdleConnDuration: 5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
		},
		"fallback": {
			Addr:                "payment-processor-fallback:8080",
			MaxConns:            1024,
			MaxIdleConnDuration: 5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
		},
	}

	// healthCheck := redis_impl.NewHealthCheckService(redisClient, cfg)
	// go healthCheck.Start()

	repo := redis_impl.NewRedisPaymentRepository(redisClient)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		worker := &redis_impl.Worker{
			Client:      redisClient,
			HostClients: clients,
			Repo:        repo,
			WorkerNum:   i,
		}
		wg.Add(1)
		go func(w *redis_impl.Worker, id int) {
			defer wg.Done()
			w.Start(ctx)
			log.Printf("Worker %d encerrado.", id)
		}(worker, i+1)
	}

	log.Println("Workers rodando. Pressione Ctrl+C para encerrar.")
	<-ctx.Done()

	log.Println("Encerrando workers...")
	wg.Wait()
	log.Println("Workers finalizados.")
}

func warmUp(redisClient *redis.Client) {
	ctx := context.Background()
	_ = redisClient.Ping(ctx).Err()
}
