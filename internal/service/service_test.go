package service

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/models"
)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	dbPath := "./test_" + time.Now().Format("20060102150405") + ".db"
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestGetEligibleOffers_UserQualifies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(db)
	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	// Create an active offer
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

	if err := svc.CreateOffer(offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Create 3 matching transactions within the lookback window
	transactions := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 1000,
			ApprovedAt:  time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 2000,
			ApprovedAt:  time.Date(2025, 10, 19, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  uuid.New().String(),
			MCC:         "5814", // Matches via MCC whitelist
			AmountCents: 1500,
			ApprovedAt:  time.Date(2025, 10, 18, 8, 0, 0, 0, time.UTC),
		},
	}

	_, err := svc.CreateTransactions(transactions)
	if err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	// Check eligibility
	response, err := svc.GetEligibleOffers(userID, now)
	if err != nil {
		t.Fatalf("Failed to get eligible offers: %v", err)
	}

	if len(response.EligibleOffers) != 1 {
		t.Fatalf("Expected 1 eligible offer, got %d", len(response.EligibleOffers))
	}

	if response.EligibleOffers[0].OfferID != offerID {
		t.Errorf("Expected %s, got %s", offerID, response.EligibleOffers[0].OfferID)
	}
}

func TestGetEligibleOffers_UserDoesNotQualify_NotEnoughTransactions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(db)
	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	// Create an active offer requiring 3 transactions
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

	if err := svc.CreateOffer(offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Create only 2 matching transactions (need 3)
	transactions := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 1000,
			ApprovedAt:  time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 2000,
			ApprovedAt:  time.Date(2025, 10, 19, 10, 0, 0, 0, time.UTC),
		},
	}

	_, err := svc.CreateTransactions(transactions)
	if err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	// Check eligibility
	response, err := svc.GetEligibleOffers(userID, now)
	if err != nil {
		t.Fatalf("Failed to get eligible offers: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Fatalf("Expected 0 eligible offers, got %d", len(response.EligibleOffers))
	}
}

func TestGetEligibleOffers_UserDoesNotQualify_OfferInactive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(db)
	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	// Create an inactive offer
	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812"},
		Active:       false, // Inactive
		MinTxnCount:  1,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := svc.CreateOffer(offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Create matching transactions
	transactions := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 1000,
			ApprovedAt:  time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC),
		},
	}

	_, err := svc.CreateTransactions(transactions)
	if err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	// Check eligibility
	response, err := svc.GetEligibleOffers(userID, now)
	if err != nil {
		t.Fatalf("Failed to get eligible offers: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Fatalf("Expected 0 eligible offers (offer is inactive), got %d", len(response.EligibleOffers))
	}
}

func TestGetEligibleOffers_UserDoesNotQualify_OutOfTimeWindow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(db)
	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	offerID := uuid.New().String()
	merchantID := uuid.New().String()
	userID := uuid.New().String()

	// Create an active offer
	offer := models.Offer{
		ID:           offerID,
		MerchantID:   merchantID,
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  1,
		LookbackDays: 7, // Only 7 days lookback
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := svc.CreateOffer(offer); err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Create a transaction that's too old (outside 7-day window)
	transactions := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchantID,
			MCC:         "5812",
			AmountCents: 1000,
			ApprovedAt:  time.Date(2025, 10, 10, 12, 0, 0, 0, time.UTC), // 11 days ago
		},
	}

	_, err := svc.CreateTransactions(transactions)
	if err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	// Check eligibility
	response, err := svc.GetEligibleOffers(userID, now)
	if err != nil {
		t.Fatalf("Failed to get eligible offers: %v", err)
	}

	if len(response.EligibleOffers) != 0 {
		t.Fatalf("Expected 0 eligible offers (transaction outside lookback window), got %d", len(response.EligibleOffers))
	}
}

func TestGetEligibleOffers_MultipleOffers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(db)
	now := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	offer1ID := uuid.New().String()
	merchant1ID := uuid.New().String()
	offer2ID := uuid.New().String()
	merchant2ID := uuid.New().String()
	userID := uuid.New().String()

	// Create two active offers
	offer1 := models.Offer{
		ID:           offer1ID,
		MerchantID:   merchant1ID,
		MCCWhitelist: []string{"5812"},
		Active:       true,
		MinTxnCount:  1,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	offer2 := models.Offer{
		ID:           offer2ID,
		MerchantID:   merchant2ID,
		MCCWhitelist: []string{"5814"},
		Active:       true,
		MinTxnCount:  2,
		LookbackDays: 30,
		StartsAt:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:       time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := svc.CreateOffer(offer1); err != nil {
		t.Fatalf("Failed to create offer1: %v", err)
	}

	if err := svc.CreateOffer(offer2); err != nil {
		t.Fatalf("Failed to create offer2: %v", err)
	}

	// Create transactions matching both offers
	transactions := []models.Transaction{
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchant1ID,
			MCC:         "5812",
			AmountCents: 1000,
			ApprovedAt:  time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  merchant2ID,
			MCC:         "5814",
			AmountCents: 2000,
			ApprovedAt:  time.Date(2025, 10, 19, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.New().String(),
			UserID:      userID,
			MerchantID:  uuid.New().String(),
			MCC:         "5814", // Matches offer2 via MCC
			AmountCents: 1500,
			ApprovedAt:  time.Date(2025, 10, 18, 8, 0, 0, 0, time.UTC),
		},
	}

	_, err := svc.CreateTransactions(transactions)
	if err != nil {
		t.Fatalf("Failed to create transactions: %v", err)
	}

	// Check eligibility
	response, err := svc.GetEligibleOffers(userID, now)
	if err != nil {
		t.Fatalf("Failed to get eligible offers: %v", err)
	}

	if len(response.EligibleOffers) != 2 {
		t.Fatalf("Expected 2 eligible offers, got %d", len(response.EligibleOffers))
	}

	// Verify both offers are present
	foundOfferIDs := make(map[string]bool)
	for _, eo := range response.EligibleOffers {
		foundOfferIDs[eo.OfferID] = true
	}

	if !foundOfferIDs[offer1ID] {
		t.Error("Expected offer1 to be eligible")
	}
	if !foundOfferIDs[offer2ID] {
		t.Error("Expected offer2 to be eligible")
	}
}

