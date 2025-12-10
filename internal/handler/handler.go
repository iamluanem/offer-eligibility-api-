package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"offer-eligibility-api/internal/models"
	"offer-eligibility-api/internal/service"
	"offer-eligibility-api/internal/validation"
)

// Handler provides HTTP handlers for the API.
type Handler struct {
	service         *service.Service
	maxBodySize     int64
}

// NewHandlerOptions holds options for creating a handler.
type NewHandlerOptions struct {
	MaxBodySize int64
}

// DefaultHandlerOptions returns default handler options.
func DefaultHandlerOptions() NewHandlerOptions {
	return NewHandlerOptions{
		MaxBodySize: 10 << 20, // 10MB default
	}
}

// NewHandler creates a new handler instance.
func NewHandler(svc *service.Service) *Handler {
	return NewHandlerWithOptions(svc, DefaultHandlerOptions())
}

// NewHandlerWithOptions creates a new handler instance with custom options.
func NewHandlerWithOptions(svc *service.Service, opts NewHandlerOptions) *Handler {
	return &Handler{
		service:     svc,
		maxBodySize: opts.MaxBodySize,
	}
}

// CreateOffer handles POST /offers
func (h *Handler) CreateOffer(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent abuse
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

	// Sanitize string fields
	req.ID = validation.SanitizeString(req.ID)
	req.MerchantID = validation.SanitizeString(req.MerchantID)
	for i := range req.MCCWhitelist {
		req.MCCWhitelist[i] = validation.SanitizeString(req.MCCWhitelist[i])
	}

	if err := h.service.CreateOffer(req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, req)
}

// CreateTransactions handles POST /transactions
func (h *Handler) CreateTransactions(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent abuse
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

	// Sanitize all transaction fields
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

// GetEligibleOffers handles GET /users/{user_id}/eligible-offers
func (h *Handler) GetEligibleOffers(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	userID = validation.SanitizeString(userID)
	
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Parse optional 'now' query parameter
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

// respondJSON sends a JSON response with the given status code.
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response with the given status code and message.
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, models.ErrorResponse{Error: message})
}

