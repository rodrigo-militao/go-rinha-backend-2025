package main

import (
	"rinha-golang/internal/config"
	"rinha-golang/internal/infra/http"
)

func main() {
	cfg := config.Load()
	http.StartServer(cfg)
}
