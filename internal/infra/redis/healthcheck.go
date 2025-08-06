package redis

import (
	"context"
	"rinha-golang/internal/config"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/redis/go-redis/v9"
)

type ProcessorStatus struct {
	URL     string
	Service string
}

type HealthCheckService struct {
	redisClient *redis.Client
	config      config.Config
	Healthy     atomic.Value
}

func NewHealthCheckService(client *redis.Client, cfg config.Config) *HealthCheckService {
	h := &HealthCheckService{redisClient: client, config: cfg}
	h.Healthy.Store(ProcessorStatus{URL: cfg.ProcessorDefaultURL, Service: "default"})
	return h
}

func (h *HealthCheckService) Start() {
	h.updateHealthyProcessor()

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
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ok, _ := h.redisClient.SetNX(ctx, "health_check_lock", "locked", 6*time.Second).Result()
	return ok
}

func (h *HealthCheckService) releaseLock() {
	h.redisClient.Del(context.Background(), "health_check_lock")
}

func (h *HealthCheckService) updateHealthyProcessor() {
	defaultURL := h.config.ProcessorDefaultURL
	serviceHealthPath := "/payments/service-health"

	if h.isHealthy(defaultURL + serviceHealthPath) {
		h.Healthy.Store(ProcessorStatus{URL: defaultURL + "/payments", Service: "default"})
		h.saveHealthyToRedis("default")
		return
	}

	h.Healthy.Store(ProcessorStatus{URL: h.config.ProcessorFallbackURL + "/payments", Service: "fallback"})
	h.saveHealthyToRedis("fallback")
}

func (h *HealthCheckService) isHealthy(url string) bool {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodGet)

	if err := fasthttp.Do(req, resp); err != nil {
		return false
	}

	if resp.StatusCode() >= 400 {
		return false
	}

	var res HealthResponse
	if err := res.UnmarshalJSON(resp.Body()); err != nil {
		return false
	}
	return !res.Failing
}

func (h *HealthCheckService) saveHealthyToRedis(service string) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	h.redisClient.HMSet(ctx, "healthy_processor_status", map[string]interface{}{
		"service":   service,
		"timestamp": time.Now().Unix(),
	})
}

func (h *HealthCheckService) readHealthyFromRedis() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
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
