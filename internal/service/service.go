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

type Service struct {
	db     *database.DB
	events *events.Manager
}

func NewService(db *database.DB) *Service {
	return &Service{
		db:     db,
		events: nil,
	}
}

func (s *Service) SetEventManager(em *events.Manager) {
	s.events = em
}

func (s *Service) CreateOffer(ctx context.Context, offer models.Offer) error {
	if err := validation.ValidateOffer(offer); err != nil {
		return err
	}

	if err := s.db.UpsertOffer(offer); err != nil {
		return err
	}

	if s.events != nil {
		s.events.PublishOfferCreated(ctx, offer)
	}

	return nil
}

func (s *Service) CreateTransactions(ctx context.Context, transactions []models.Transaction) (int, error) {
	if len(transactions) == 0 {
		return 0, fmt.Errorf("no transactions provided")
	}

	if len(transactions) > 1000 {
		return 0, fmt.Errorf("cannot process more than 1000 transactions per request")
	}

	for i, txn := range transactions {
		if err := validation.ValidateTransaction(txn); err != nil {
			return 0, fmt.Errorf("invalid transaction at index %d: %w", i, err)
		}
	}

	count, err := s.db.InsertTransactions(transactions)
	if err != nil {
		return 0, err
	}

	if s.events != nil {
		s.events.PublishTransactionCreated(ctx, transactions, count)
	}

	return count, nil
}

func (s *Service) GetEligibleOffers(ctx context.Context, userID string, now time.Time) (models.EligibleOffersResponse, error) {
	if err := validation.ValidateUUID(userID, "user_id"); err != nil {
		return models.EligibleOffersResponse{}, err
	}

	activeOffers, err := s.db.GetActiveOffers(now)
	if err != nil {
		return models.EligibleOffersResponse{}, fmt.Errorf("failed to get active offers: %w", err)
	}

	var eligibleOffers []models.EligibleOffer

	for _, offer := range activeOffers {
		matchCount, err := s.db.CountMatchingTransactions(userID, offer, now)
		if err != nil {
			return models.EligibleOffersResponse{}, fmt.Errorf("failed to count transactions: %w", err)
		}

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

	if s.events != nil {
		s.events.PublishEligibilityChecked(ctx, userID, eligibleOffers)
	}

	return response, nil
}
