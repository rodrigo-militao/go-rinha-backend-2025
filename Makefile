# Makefile para profiling da Rinha

NOW := $(shell date +"%Y%m%d-%H%M%S")
K6_SCRIPT := ~/dev/rinha-de-backend-2025/rinha-test/rinha.js

.PHONY: all profile up down clean

all: profile

up:
	@echo "🧹 Limpando containers anteriores..."
	docker compose down -v

	@echo "🚀 Subindo containers com build forçado..."
	docker compose up --build -d

	@echo "⏳ Aguardando serviços responderem..."
	@until curl -s http://localhost:6061/debug/pprof/ > /dev/null; do echo "⌛ Esperando backend-1..."; sleep 1; done
# 	@until curl -s http://localhost:6063/debug/pprof/ > /dev/null; do echo "⌛ Esperando worker..."; sleep 1; done

profile: up
	@echo "📈 Iniciando coleta de profile em paralelo..."
	go tool pprof -pdf http://localhost:6061/debug/pprof/profile?seconds=30 > profile-backend-1-$(NOW).pdf &
	go tool pprof -pdf http://localhost:6062/debug/pprof/profile?seconds=30 > profile-backend-2-$(NOW).pdf &
# 	go tool pprof -pdf http://localhost:6063/debug/pprof/profile?seconds=30 > profile-worker-$(NOW).pdf &

	@echo "🧪 Executando teste de carga com K6..."
	k6 run $(K6_SCRIPT)

	@echo "⏳ Aguardando finalização da coleta de perfis..."
	wait

	@echo "✅ Perfis gerados com sucesso:"
	@ls -lh profile-*$(NOW).pdf

	@$(MAKE) down

down:
	@echo "🛑 Finalizando containers..."
	docker compose down -v

clean:
	@echo "🧽 Removendo perfis antigos..."
	rm -f profile-*.pdf
