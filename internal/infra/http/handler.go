package http

import (
	"encoding/json"
	"log"
	"net/http"
	"rinha-golang/internal/application"
	"rinha-golang/internal/domain"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	ProcessPaymentUC *application.ProcessPaymentUseCase
	GetSummaryUC     *application.GetSummaryUseCase
}

type paymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

func (h *Handler) HandlePayments(w http.ResponseWriter, r *http.Request) {
	var req paymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler] Invalid body: %v", err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(req.CorrelationId)
	if err != nil {
		log.Printf("[Handler] Invalid correlationId: %v", err)
		http.Error(w, "invalid correlationId", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		log.Printf("[Handler] Invalid amount: %v", req.Amount)
		http.Error(w, "amount must be > 0", http.StatusBadRequest)
		return
	}
	payment := domain.Payment{
		CorrelationId: id,
		Amount:        req.Amount,
		RequestedAt:   time.Now().UTC(),
	}

	err = h.ProcessPaymentUC.Execute(r.Context(), payment)
	if err != nil {
		log.Printf("[Handler] Failed to enqueue payment: %v", err)
		http.Error(w, "failed to enqueue payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

}

func (h *Handler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
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
		log.Printf("[Handler] Failed to get summary: %v", err)
		http.Error(w, "failed to get summary", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(summary); err != nil {
		log.Printf("[Handler] Failed to encode summary: %v", err)
		http.Error(w, "failed to encode summary", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) PurgePayments(w http.ResponseWriter, r *http.Request) {
	h.ProcessPaymentUC.PurgePayments(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
