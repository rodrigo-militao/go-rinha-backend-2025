package redis

import (
	"context"
	"net/http"
	"rinha-golang/internal/config"
	"sync/atomic"
	"time"

	json "github.com/json-iterator/go"

	"github.com/redis/go-redis/v9"
)

type ProcessorStatus struct {
	URL     string
	Service string
}

type HealthCheckService struct {
	redisClient *redis.Client
	config      config.Config
	Healthy     atomic.Value // ProcessorStatus
}

func NewHealthCheckService(client *redis.Client, cfg config.Config) *HealthCheckService {
	h := &HealthCheckService{redisClient: client, config: cfg}
	h.Healthy.Store(ProcessorStatus{URL: cfg.ProcessorDefaultURL, Service: "default"})
	return h
}

func (h *HealthCheckService) Start() {
	go func() {
		ticker := time.NewTicker(6 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if h.acquireLock() {
				h.updateHealthyProcessor()
				h.releaseLock()
			} else {
				h.readHealthyFromRedis()
			}
		}
	}()
}

func (h *HealthCheckService) acquireLock() bool {
	ctx := context.Background()
	ok, _ := h.redisClient.SetNX(ctx, "health_check_lock", "locked", 10*time.Second).Result()
	return ok
}

func (h *HealthCheckService) releaseLock() {
	h.redisClient.Del(context.Background(), "health_check_lock")
}

func (h *HealthCheckService) updateHealthyProcessor() {
	defaultURL := h.config.ProcessorDefaultURL
	fallbackURL := h.config.ProcessorFallbackURL
	serviceHealthPath := "/service-health"

	if h.isHealthy(defaultURL + serviceHealthPath) {
		h.Healthy.Store(ProcessorStatus{URL: defaultURL + "/payments", Service: "default"})
		h.saveHealthyToRedis("default")
		return
	}
	if h.isHealthy(fallbackURL + serviceHealthPath) {
		h.Healthy.Store(ProcessorStatus{URL: fallbackURL + "/payments", Service: "fallback"})
		h.saveHealthyToRedis("fallback")
		return
	}
	cur := h.Healthy.Load().(ProcessorStatus)
	h.saveHealthyToRedis(cur.Service)
}

func (h *HealthCheckService) isHealthy(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var res struct{ Failing bool }
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return !res.Failing
}

func (h *HealthCheckService) saveHealthyToRedis(service string) {
	ctx := context.Background()
	h.redisClient.HMSet(ctx, "healthy_processor_status", map[string]interface{}{
		"service":   service,
		"timestamp": time.Now().Unix(),
	})
}

func (h *HealthCheckService) readHealthyFromRedis() {
	ctx := context.Background()
	m, _ := h.redisClient.HGetAll(ctx, "healthy_processor_status").Result()
	switch m["service"] {
	case "default":
		h.Healthy.Store(ProcessorStatus{URL: h.config.ProcessorDefaultURL + "/payments", Service: "default"})
	case "fallback":
		h.Healthy.Store(ProcessorStatus{URL: h.config.ProcessorFallbackURL + "/payments", Service: "fallback"})
	}
}

func (h *HealthCheckService) GetCurrent() ProcessorStatus {
	return h.Healthy.Load().(ProcessorStatus)
}
