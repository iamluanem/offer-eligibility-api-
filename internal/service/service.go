package service

import (
	"fmt"
	"time"

	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/models"
)

// Service provides business logic for the offer eligibility API.
type Service struct {
	db *database.DB
}

// NewService creates a new service instance.
func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

// CreateOffer creates or updates an offer.
func (s *Service) CreateOffer(offer models.Offer) error {
	if err := s.validateOffer(offer); err != nil {
		return err
	}

	return s.db.UpsertOffer(offer)
}

// CreateTransactions ingests multiple transactions.
func (s *Service) CreateTransactions(transactions []models.Transaction) (int, error) {
	if len(transactions) == 0 {
		return 0, fmt.Errorf("no transactions provided")
	}

	// Validate all transactions before inserting
	for _, txn := range transactions {
		if err := s.validateTransaction(txn); err != nil {
			return 0, fmt.Errorf("invalid transaction %s: %w", txn.ID, err)
		}
	}

	return s.db.InsertTransactions(transactions)
}

// GetEligibleOffers returns all offers that a user is eligible for at the given time.
func (s *Service) GetEligibleOffers(userID string, now time.Time) (models.EligibleOffersResponse, error) {
	if userID == "" {
		return models.EligibleOffersResponse{}, fmt.Errorf("user_id is required")
	}

	// Get all active offers at the current time
	activeOffers, err := s.db.GetActiveOffers(now)
	if err != nil {
		return models.EligibleOffersResponse{}, fmt.Errorf("failed to get active offers: %w", err)
	}

	var eligibleOffers []models.EligibleOffer

	for _, offer := range activeOffers {
		// Count matching transactions for this user and offer
		matchCount, err := s.db.CountMatchingTransactions(userID, offer, now)
		if err != nil {
			return models.EligibleOffersResponse{}, fmt.Errorf("failed to count transactions: %w", err)
		}

		// Check if user meets the minimum transaction count requirement
		if matchCount >= offer.MinTxnCount {
			reason := fmt.Sprintf(">= %d matching transactions in last %d days (found %d)",
				offer.MinTxnCount, offer.LookbackDays, matchCount)
			eligibleOffers = append(eligibleOffers, models.EligibleOffer{
				OfferID: offer.ID,
				Reason:  reason,
			})
		}
	}

	return models.EligibleOffersResponse{
		UserID:         userID,
		EligibleOffers: eligibleOffers,
	}, nil
}

// validateOffer performs basic validation on an offer.
func (s *Service) validateOffer(offer models.Offer) error {
	if offer.ID == "" {
		return fmt.Errorf("offer id is required")
	}
	if offer.MerchantID == "" {
		return fmt.Errorf("merchant_id is required")
	}
	if offer.StartsAt.After(offer.EndsAt) {
		return fmt.Errorf("starts_at must be before ends_at")
	}
	if offer.MinTxnCount < 0 {
		return fmt.Errorf("min_txn_count must be non-negative")
	}
	if offer.LookbackDays < 0 {
		return fmt.Errorf("lookback_days must be non-negative")
	}
	return nil
}

// validateTransaction performs basic validation on a transaction.
func (s *Service) validateTransaction(txn models.Transaction) error {
	if txn.ID == "" {
		return fmt.Errorf("transaction id is required")
	}
	if txn.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if txn.MerchantID == "" {
		return fmt.Errorf("merchant_id is required")
	}
	if txn.MCC == "" {
		return fmt.Errorf("mcc is required")
	}
	if txn.AmountCents < 0 {
		return fmt.Errorf("amount_cents must be non-negative")
	}
	return nil
}

