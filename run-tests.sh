#!/bin/bash

set -e

# Parar e remover tudo (inclusive volumes)
echo "🧹 Limpando containers anteriores..."
docker compose down -v

# Subir tudo com build forçado
echo "🚀 Subindo containers com build..."
docker compose up --build -d

# Aguardar alguns segundos para garantir que os serviços estão prontos
echo "⏳ Aguardando estabilização dos serviços..."
sleep 5

# Executar os testes K6
echo "🧪 Rodando testes de performance com K6..."
k6 run ~/dev/rinha-de-backend-2025/rinha-test/rinha-final.js

# echo ">>> Capturando profile do backend-1"
# go tool pprof -pdf http://localhost:6061/debug/pprof/profile?seconds=30 > profile-backend-1.pdf

# echo ">>> Capturando profile do worker"
# go tool pprof -pdf http://localhost:6063/debug/pprof/profile?seconds=30 > profile-worker.pdf

# echo ">>> Análise completa em arquivos PDF gerados."

echo "🛑 Finalizando containers..."
docker compose down -v
