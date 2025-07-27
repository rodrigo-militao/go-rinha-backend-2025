package main

import (
	"context"
	"fmt"
	"log"
	"net"

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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Network: "unix",
		Addr:    "/tmp/redis.sock",
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
		worker := &redis_impl.Worker{Client: redisClient, Health: healthCheck, Repo: paymentRepo, WorkerNum: i}
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker.Start(ctx)
			log.Printf("Worker %d encerrado.", workerID)
		}(i + 1)
	}

	routes_handler := http_infra.SetupRoutes(processPaymentUC, getSummaryUC)

	socketPath := fmt.Sprintf("/tmp/app-sockets/%s.sock", cfg.InstanceID)
	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Falha ao escutar no unix socket: %v", err)
	}

	if err := os.Chmod(socketPath, 0777); err != nil {
		log.Fatalf("Falha ao alterar permissões do socket: %v", err)
	}

	server := &fasthttp.Server{
		Handler:           routes_handler,
		Name:              "rinha-go",
		ReduceMemoryUsage: true,
		Concurrency:       256 * 1024,
	}

	go func() {
		log.Printf("Servidor HTTP escutando no socket %s", socketPath)
		if err := server.Serve(unixListener); err != nil {
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
