package service

import (
	"context"
	"fmt"
	"time"

	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/events"
	"offer-eligibility-api/internal/models"
	"offer-eligibility-api/internal/validation"
)

// Service provides business logic for the offer eligibility API.
type Service struct {
	db     *database.DB
	events *events.Manager
}

// NewService creates a new service instance.
func NewService(db *database.DB) *Service {
	return &Service{
		db:     db,
		events: nil, // Will be set via SetEventManager if needed
	}
}

// SetEventManager sets the event manager for the service.
func (s *Service) SetEventManager(em *events.Manager) {
	s.events = em
}

// CreateOffer creates or updates an offer.
func (s *Service) CreateOffer(ctx context.Context, offer models.Offer) error {
	if err := validation.ValidateOffer(offer); err != nil {
		return err
	}

	if err := s.db.UpsertOffer(offer); err != nil {
		return err
	}

	// Publish event
	if s.events != nil {
		s.events.PublishOfferCreated(ctx, offer)
	}

	return nil
}

// CreateTransactions ingests multiple transactions.
func (s *Service) CreateTransactions(ctx context.Context, transactions []models.Transaction) (int, error) {
	if len(transactions) == 0 {
		return 0, fmt.Errorf("no transactions provided")
	}

	if len(transactions) > 1000 {
		return 0, fmt.Errorf("cannot process more than 1000 transactions per request")
	}

	// Validate all transactions before inserting
	for i, txn := range transactions {
		if err := validation.ValidateTransaction(txn); err != nil {
			return 0, fmt.Errorf("invalid transaction at index %d: %w", i, err)
		}
	}

	count, err := s.db.InsertTransactions(transactions)
	if err != nil {
		return 0, err
	}

	// Publish event
	if s.events != nil {
		s.events.PublishTransactionCreated(ctx, transactions, count)
	}

	return count, nil
}

// GetEligibleOffers returns all offers that a user is eligible for at the given time.
func (s *Service) GetEligibleOffers(ctx context.Context, userID string, now time.Time) (models.EligibleOffersResponse, error) {
	if err := validation.ValidateUUID(userID, "user_id"); err != nil {
		return models.EligibleOffersResponse{}, err
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

	response := models.EligibleOffersResponse{
		UserID:         userID,
		EligibleOffers: eligibleOffers,
	}

	// Publish event
	if s.events != nil {
		s.events.PublishEligibilityChecked(ctx, userID, eligibleOffers)
	}

	return response, nil
}
