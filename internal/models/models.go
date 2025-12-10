package models

import "time"

type Offer struct {
	ID           string    `json:"id"`
	MerchantID   string    `json:"merchant_id"`
	MCCWhitelist []string  `json:"mcc_whitelist"`
	Active       bool      `json:"active"`
	MinTxnCount  int       `json:"min_txn_count"`
	LookbackDays int       `json:"lookback_days"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
}

type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	MerchantID  string    `json:"merchant_id"`
	MCC         string    `json:"mcc"`
	AmountCents int64     `json:"amount_cents"`
	ApprovedAt  time.Time `json:"approved_at"`
}

type EligibleOffer struct {
	OfferID string `json:"offer_id"`
	Reason  string `json:"reason"`
}

type EligibleOffersResponse struct {
	UserID         string          `json:"user_id"`
	EligibleOffers []EligibleOffer `json:"eligible_offers"`
}

type CreateOfferRequest struct {
	Offer Offer `json:"offer"`
}

type CreateTransactionsRequest struct {
	Transactions []Transaction `json:"transactions"`
}

type CreateTransactionsResponse struct {
	Inserted int `json:"inserted"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
