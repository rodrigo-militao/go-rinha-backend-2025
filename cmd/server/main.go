package main

import (
	"rinha-golang/internal/config"
	"rinha-golang/internal/config/infra/http"
)

func main() {
	cfg := config.Load()
	http.StartServer(cfg)
}
