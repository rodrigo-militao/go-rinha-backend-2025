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
k6 run ~/dev/rinha-de-backend-2025/rinha-test/rinha.js

# Parar tudo após os testes (opcional)
echo "🛑 Finalizando containers..."
docker compose down -v
