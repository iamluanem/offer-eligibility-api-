package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"offer-eligibility-api/internal/models"
	"offer-eligibility-api/internal/service"
	"offer-eligibility-api/internal/validation"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service     *service.Service
	maxBodySize int64
}

type NewHandlerOptions struct {
	MaxBodySize int64
}

func DefaultHandlerOptions() NewHandlerOptions {
	return NewHandlerOptions{
		MaxBodySize: 10 << 20,
	}
}

func NewHandler(svc *service.Service) *Handler {
	return NewHandlerWithOptions(svc, DefaultHandlerOptions())
}

func NewHandlerWithOptions(svc *service.Service, opts NewHandlerOptions) *Handler {
	return &Handler{
		service:     svc,
		maxBodySize: opts.MaxBodySize,
	}
}

func (h *Handler) CreateOffer(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodySize)

	var req models.Offer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			h.respondError(w, http.StatusBadRequest, "request body is required")
			return
		}
		h.respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}

	req.ID = validation.SanitizeString(req.ID)
	req.MerchantID = validation.SanitizeString(req.MerchantID)
	for i := range req.MCCWhitelist {
		req.MCCWhitelist[i] = validation.SanitizeString(req.MCCWhitelist[i])
	}

	if err := h.service.CreateOffer(r.Context(), req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, req)
}

func (h *Handler) CreateTransactions(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodySize)

	var req models.CreateTransactionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			h.respondError(w, http.StatusBadRequest, "request body is required")
			return
		}
		h.respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}

	for i := range req.Transactions {
		txn := &req.Transactions[i]
		txn.ID = validation.SanitizeString(txn.ID)
		txn.UserID = validation.SanitizeString(txn.UserID)
		txn.MerchantID = validation.SanitizeString(txn.MerchantID)
		txn.MCC = validation.SanitizeString(txn.MCC)
	}

	inserted, err := h.service.CreateTransactions(r.Context(), req.Transactions)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, models.CreateTransactionsResponse{
		Inserted: inserted,
	})
}

func (h *Handler) GetEligibleOffers(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	userID = validation.SanitizeString(userID)

	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	now := time.Now().UTC()
	if nowParam := r.URL.Query().Get("now"); nowParam != "" {
		nowParam = validation.SanitizeString(nowParam)
		parsed, err := validation.ValidateTimeString(nowParam)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid 'now' parameter, must be RFC3339 format")
			return
		}
		now = parsed.UTC()
	}

	response, err := h.service.GetEligibleOffers(r.Context(), userID, now)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, models.ErrorResponse{Error: message})
}
