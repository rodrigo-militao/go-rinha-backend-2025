package http

import (
	"log"
	"rinha-golang/internal/application"
	"rinha-golang/internal/domain"
	"time"

	json "github.com/json-iterator/go"

	"github.com/valyala/fasthttp"
)

type Handler struct {
	ProcessPaymentUC *application.ProcessPaymentUseCase
	GetSummaryUC     *application.GetSummaryUseCase
}

type paymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

func (h *Handler) HandlePayments(ctx *fasthttp.RequestCtx) {
	var req paymentRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error("invalid body", fasthttp.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		ctx.Error("invalid body", fasthttp.StatusBadRequest)
		return
	}

	payment := domain.Payment{
		CorrelationId: req.CorrelationId,
		Amount:        req.Amount,
		RequestedAt:   time.Now().UTC(),
	}

	err := h.ProcessPaymentUC.Execute(ctx, payment)
	if err != nil {
		log.Printf("[Handler] Failed to enqueue payment: %v", err)
		ctx.Error("failed to process payment", fasthttp.StatusServiceUnavailable)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
}

func (h *Handler) HandleSummary(ctx *fasthttp.RequestCtx) {
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))
	var from, to *time.Time

	if fromStr != "" {
		f, err := time.Parse(time.RFC3339, fromStr)
		if err == nil {
			from = &f
		}
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err == nil {
			to = &t
		}
	}

	summary, err := h.GetSummaryUC.Execute(from, to)
	if err != nil {
		ctx.Error("failed to get summary", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	if err := json.NewEncoder(ctx).Encode(summary); err != nil {
		log.Printf("Failed to encode summary: %v", err)
	}
}

func (h *Handler) HandleHealth(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write([]byte("ok"))
}

func (h *Handler) PurgePayments(ctx *fasthttp.RequestCtx) {
	h.ProcessPaymentUC.PurgePayments(ctx)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
}
