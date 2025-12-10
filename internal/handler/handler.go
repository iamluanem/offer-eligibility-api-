package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"offer-eligibility-api/internal/models"
	"offer-eligibility-api/internal/service"
)

// Handler provides HTTP handlers for the API.
type Handler struct {
	service *service.Service
}

// NewHandler creates a new handler instance.
func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

// CreateOffer handles POST /offers
func (h *Handler) CreateOffer(w http.ResponseWriter, r *http.Request) {
	var req models.Offer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CreateOffer(req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, req)
}

// CreateTransactions handles POST /transactions
func (h *Handler) CreateTransactions(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTransactionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inserted, err := h.service.CreateTransactions(req.Transactions)
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
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Parse optional 'now' query parameter
	now := time.Now().UTC()
	if nowParam := r.URL.Query().Get("now"); nowParam != "" {
		parsed, err := time.Parse(time.RFC3339, nowParam)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid 'now' parameter, must be RFC3339 format")
			return
		}
		now = parsed.UTC()
	}

	response, err := h.service.GetEligibleOffers(userID, now)
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

