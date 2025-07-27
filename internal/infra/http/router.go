package http

import (
	"rinha-golang/internal/application"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

func SetupRoutes(
	processPaymentUC *application.ProcessPaymentUseCase,
	getSummaryUC *application.GetSummaryUseCase) fasthttp.RequestHandler {

	handler := &Handler{ProcessPaymentUC: processPaymentUC, GetSummaryUC: getSummaryUC}

	router := fasthttprouter.New()
	router.POST("/payments", handler.HandlePayments)
	router.POST("/purge-payments", handler.PurgePayments)
	router.GET("/payments-summary", handler.HandleSummary)
	router.GET("/health", handler.HandleHealth)

	return router.Handler
}
