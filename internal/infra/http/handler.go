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
	Repo         domain.PaymentRepository
	GetSummaryUC *application.GetSummaryUseCase
}

type paymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

func (h *Handler) HandlePayments(ctx *fasthttp.RequestCtx) {
	body := append([]byte(nil), ctx.Request.Body()...)
	h.Repo.AddToStream(ctx, body)

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
	h.Repo.PurgePayments(ctx)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
}
