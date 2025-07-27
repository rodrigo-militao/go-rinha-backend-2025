package http

import (
	"context"
	"net/http"
	"rinha-golang/internal/application"
	"rinha-golang/internal/config"
	redis_impl "rinha-golang/internal/infra/redis"

	"github.com/redis/go-redis/v9"
)

func StartServer(cfg config.Config) {
	redisClient := redis.NewClient(&redis.Options{
		Network: "unix",
		Addr:    "/tmp/redis.sock",
	})

	paymentRepo := redis_impl.NewRedisPaymentRepository(redisClient)
	processPaymentUC := &application.ProcessPaymentUseCase{Repo: paymentRepo}
	getSummaryUC := &application.GetSummaryUseCase{Repo: paymentRepo}

	handler := &Handler{ProcessPaymentUC: processPaymentUC, GetSummaryUC: getSummaryUC}

	healthCheck := redis_impl.NewHealthCheckService(redisClient, cfg)
	healthCheck.Start()
	for i := 0; i < cfg.Workers; i++ {
		worker := &redis_impl.Worker{Client: redisClient, Health: healthCheck, Repo: paymentRepo, WorkerNum: i}
		go worker.Start(context.Background())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/payments", handler.HandlePayments)
	mux.HandleFunc("/payments-summary", handler.HandleSummary)
	mux.HandleFunc("/purge-payments", handler.PurgePayments)
	mux.HandleFunc("/health", handler.HandleHealth)

	http.ListenAndServe(cfg.ListenAddr, mux)
}
