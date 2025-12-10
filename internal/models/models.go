package models

import "time"

// Offer represents a merchant offer / promotion.
type Offer struct {
	ID           string    `json:"id"`            // uuid
	MerchantID   string    `json:"merchant_id"`   // uuid
	MCCWhitelist []string  `json:"mcc_whitelist"` // e.g. ["5812", "5814"]
	Active       bool      `json:"active"`
	MinTxnCount  int       `json:"min_txn_count"` // N
	LookbackDays int       `json:"lookback_days"` // K days
	StartsAt     time.Time `json:"starts_at"`     // RFC3339 timestamp
	EndsAt       time.Time `json:"ends_at"`       // RFC3339 timestamp
}

// Transaction represents a single user transaction.
type Transaction struct {
	ID          string    `json:"id"`           // uuid
	UserID      string    `json:"user_id"`      // uuid
	MerchantID  string    `json:"merchant_id"`  // uuid
	MCC         string    `json:"mcc"`          // 4-digit merchant category code
	AmountCents int64     `json:"amount_cents"` // integer cents
	ApprovedAt  time.Time `json:"approved_at"`  // RFC3339 timestamp
}

// EligibleOffer represents an offer that a user is eligible for.
type EligibleOffer struct {
	OfferID string `json:"offer_id"`
	Reason  string `json:"reason"` // short human explanation
}

// EligibleOffersResponse is the response payload when asking for eligibility.
type EligibleOffersResponse struct {
	UserID         string          `json:"user_id"`
	EligibleOffers []EligibleOffer `json:"eligible_offers"`
}

// CreateOfferRequest represents the request body for creating an offer.
type CreateOfferRequest struct {
	Offer Offer `json:"offer"`
}

// CreateTransactionsRequest represents the request body for ingesting transactions.
type CreateTransactionsRequest struct {
	Transactions []Transaction `json:"transactions"`
}

// CreateTransactionsResponse represents the response for ingesting transactions.
type CreateTransactionsResponse struct {
	Inserted int `json:"inserted"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

