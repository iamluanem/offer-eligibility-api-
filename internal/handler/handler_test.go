package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/models"
	"offer-eligibility-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func setupTestHandler(t *testing.T) (*Handler, func()) {
	dbPath := "./test_handler_" + time.Now().Format("20060102150405") + ".db"
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	svc := service.NewService(db)
	h := NewHandler(svc)

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return h, cleanup
}

func setupRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/offers", h.CreateOffer)
	r.Post("/transactions", h.CreateTransactions)
	r.Get("/users/{user_id}/eligible-offers", h.GetEligibleOffers)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	return r
}

func TestHealthCheck(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rr.Body.String())
	}
}

func TestCreateOffer_Success(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()

	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812", "5814"},
		Active:       true,
		MinTxnCount:  3,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	body, _ := json.Marshal(offer)
	req := httptest.NewRequest("POST", "/offers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.Offer
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ID != offerID {
		t.Errorf("Expected ID %s, got %s", offerID, response.ID)
	}
}

func TestCreateOffer_InvalidJSON(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	req := httptest.NewRequest("POST", "/offers", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Error == "" {
		t.Error("Expected error message in response")
	}
}

func TestCreateOffer_EmptyBody(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	req := httptest.NewRequest("POST", "/offers", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestCreateOffer_InvalidUUID(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offer := models.Offer{
		ID:           "invalid-uuid",
		MerchantID:   uuid.New().String(),
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  3,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	body, _ := json.Marshal(offer)
	req := httptest.NewRequest("POST", "/offers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateOffer_InvalidMCC(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offer := models.Offer{
		ID:           uuid.New().String(),
		MerchantID:   uuid.New().String(),
		MCCWhitelist: []string{"123"},
		Active:       true,
		MinTxnCount:  3,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	body, _ := json.Marshal(offer)
	req := httptest.NewRequest("POST", "/offers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateOffer_InvalidTimeRange(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offer := models.Offer{
		ID:           uuid.New().String(),
		MerchantID:   uuid.New().String(),
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  3,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
	}

	body, _ := json.Marshal(offer)
	req := httptest.NewRequest("POST", "/offers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateTransactions_Success(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	userID := uuid.New().String()
	merchantID := uuid.New().String()

	reqBody := models.CreateTransactionsRequest{
		Transactions: []models.Transaction{
			{
				ID:          uuid.New().String(),
				UserID:      userID,
				MerchantID:  merchantID,
				MCC:         "5812",
				AmountCents: 1250,
				ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
			},
			{
				ID:          uuid.New().String(),
				UserID:      userID,
				MerchantID:  merchantID,
				MCC:         "5812",
				AmountCents: 890,
				ApprovedAt:  time.Date(2025, 10, 19, 13, 10, 0, 0, time.UTC),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.CreateTransactionsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Inserted != 2 {
		t.Errorf("Expected 2 inserted, got %d", response.Inserted)
	}
}

func TestCreateTransactions_EmptyBody(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	req := httptest.NewRequest("POST", "/transactions", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestCreateTransactions_InvalidJSON(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	req := httptest.NewRequest("POST", "/transactions", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestCreateTransactions_InvalidUserID(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	reqBody := models.CreateTransactionsRequest{
		Transactions: []models.Transaction{
			{
				ID:          uuid.New().String(),
				UserID:      "invalid-uuid",
				MerchantID:  uuid.New().String(),
				MCC:         "5812",
				AmountCents: 1250,
				ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateTransactions_InvalidMCC(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	reqBody := models.CreateTransactionsRequest{
		Transactions: []models.Transaction{
			{
				ID:          uuid.New().String(),
				UserID:      uuid.New().String(),
				MerchantID:  uuid.New().String(),
				MCC:         "12",
				AmountCents: 1250,
				ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateTransactions_MissingApprovedAt(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	reqBody := models.CreateTransactionsRequest{
		Transactions: []models.Transaction{
			{
				ID:          uuid.New().String(),
				UserID:      uuid.New().String(),
				MerchantID:  uuid.New().String(),
				MCC:         "5812",
				AmountCents: 1250,
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateTransactions_DuplicateID(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	txnID := uuid.New().String()
	userID := uuid.New().String()
	merchantID := uuid.New().String()

	reqBody := models.CreateTransactionsRequest{
		Transactions: []models.Transaction{
			{
				ID:          txnID,
				UserID:      userID,
				MerchantID:  merchantID,
				MCC:         "5812",
				AmountCents: 1250,
				ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("First insert failed: %d. Body: %s", rr.Code, rr.Body.String())
	}

	req2 := httptest.NewRequest("POST", "/transactions", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate, got %d. Body: %s", rr2.Code, rr2.Body.String())
	}
}

func TestGetEligibleOffers_Success(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812", "5814"},
		Active:       true,
		MinTxnCount:  3,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := h.service.CreateOffer(context.Background(), offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	txns := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 1250,
			ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 890,
			ApprovedAt:  time.Date(2025, 10, 19, 13, 10, 0, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  uuid.New().String(),
			MCC:         "5814",
			AmountCents: 1500,
			ApprovedAt:  time.Date(2025, 10, 18, 10, 0, 0, 0, time.UTC),
		},
	}

	if _, err := h.service.CreateTransactions(context.Background(), txns); err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers?now="+now.Format(time.RFC3339), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.EligibleOffersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UserID != userID {
		t.Errorf("Expected user_id %s, got %s", userID, response.UserID)
	}

	if len(response.EligibleOffers) != 1 {
		t.Errorf("Expected 1 eligible offer, got %d", len(response.EligibleOffers))
	}

	if response.EligibleOffers[0].OfferID != offerID {
		t.Errorf("Expected offer_id %s, got %s", offerID, response.EligibleOffers[0].OfferID)
	}
}

func TestGetEligibleOffers_NoEligibleOffers(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	userID := uuid.New().String()

	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers?now="+now.Format(time.RFC3339), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.EligibleOffersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Errorf("Expected 0 eligible offers, got %d", len(response.EligibleOffers))
	}
}

func TestGetEligibleOffers_InvalidUserID(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	req := httptest.NewRequest("GET", "/users/invalid-uuid/eligible-offers", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetEligibleOffers_InvalidTimeParameter(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	userID := uuid.New().String()
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers?now=invalid-time", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetEligibleOffers_EmptyUserID(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	req := httptest.NewRequest("GET", "/users//eligible-offers", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected error for empty user_id")
	}
}

func TestGetEligibleOffers_WithoutTimeParameter(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	userID := uuid.New().String()
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.EligibleOffersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UserID != userID {
		t.Errorf("Expected user_id %s, got %s", userID, response.UserID)
	}
}

func TestGetEligibleOffers_InactiveOffer(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812"},
		Active:       false,
		MinTxnCount:  1,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := h.service.CreateOffer(context.Background(), offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	txn := models.Transaction{
		ID:          uuid.New().String(),
		UserID:      userID,
		MerchantID:  merchantID,
		MCC:         "5812",
		AmountCents: 1250,
		ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
	}

	if _, err := h.service.CreateTransactions(context.Background(), []models.Transaction{txn}); err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers?now="+now.Format(time.RFC3339), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.EligibleOffersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Errorf("Expected 0 eligible offers (inactive), got %d", len(response.EligibleOffers))
	}
}

func TestGetEligibleOffers_OutOfTimeWindow(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  1,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 5, 23, 59, 59, 0, time.UTC),
	}

	if err := h.service.CreateOffer(context.Background(), offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	txn := models.Transaction{
		ID:          uuid.New().String(),
		UserID:      userID,
		MerchantID:  merchantID,
		MCC:         "5812",
		AmountCents: 1250,
		ApprovedAt:  time.Date(2025, 10, 4, 12, 34, 56, 0, time.UTC),
	}

	if _, err := h.service.CreateTransactions(context.Background(), []models.Transaction{txn}); err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers?now="+now.Format(time.RFC3339), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.EligibleOffersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Errorf("Expected 0 eligible offers (out of time window), got %d", len(response.EligibleOffers))
	}
}

func TestGetEligibleOffers_NotEnoughTransactions(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  5,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := h.service.CreateOffer(context.Background(), offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	txns := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 1250,
			ApprovedAt:  time.Date(2025, 10, 20, 12, 34, 56, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 890,
			ApprovedAt:  time.Date(2025, 10, 19, 13, 10, 0, 0, time.UTC),
		},
	}

	if _, err := h.service.CreateTransactions(context.Background(), txns); err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest("GET", "/users/"+userID+"/eligible-offers?now="+now.Format(time.RFC3339), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var response models.EligibleOffersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Errorf("Expected 0 eligible offers (not enough transactions), got %d", len(response.EligibleOffers))
	}
}

func TestCreateOffer_Upsert(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	r := setupRouter(h)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()

	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  3,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	body, _ := json.Marshal(offer)
	req := httptest.NewRequest("POST", "/offers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("First create failed: %d. Body: %s", rr.Code, rr.Body.String())
	}

	offer.MCCWhitelist = []string{"5812", "5814"}
	offer.MinTxnCount = 5

	body2, _ := json.Marshal(offer)
	req2 := httptest.NewRequest("POST", "/offers", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Errorf("Upsert failed: %d. Body: %s", rr2.Code, rr2.Body.String())
	}

	var response models.Offer
	if err := json.Unmarshal(rr2.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.MinTxnCount != 5 {
		t.Errorf("Expected MinTxnCount 5 after upsert, got %d", response.MinTxnCount)
	}

	if len(response.MCCWhitelist) != 2 {
		t.Errorf("Expected 2 MCCs after upsert, got %d", len(response.MCCWhitelist))
	}
}
