package http

import (
	"rinha-golang/internal/application"
	"rinha-golang/internal/domain"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

func SetupRoutes(
	repository domain.PaymentRepository,
	getSummaryUC *application.GetSummaryUseCase,
) fasthttp.RequestHandler {

	handler := &Handler{Repo: repository, GetSummaryUC: getSummaryUC}

	router := fasthttprouter.New()
	router.POST("/payments", handler.HandlePayments)
	router.POST("/purge-payments", handler.PurgePayments)
	router.GET("/payments-summary", handler.HandleSummary)
	router.GET("/health", handler.HandleHealth)

	return router.Handler
}
