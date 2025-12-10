package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"offer-eligibility-api/internal/models"
)

var (
	// uuidRegex validates UUID format (v4)
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	// mccRegex validates 4-digit MCC codes
	mccRegex = regexp.MustCompile(`^\d{4}$`)
)

// ValidationError represents a validation error with field context.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidateOffer validates an offer and returns any validation errors.
func ValidateOffer(offer models.Offer) error {
	if err := ValidateUUID(offer.ID, "id"); err != nil {
		return err
	}

	if err := ValidateUUID(offer.MerchantID, "merchant_id"); err != nil {
		return err
	}

	if err := validateMCCWhitelist(offer.MCCWhitelist); err != nil {
		return err
	}

	if offer.MinTxnCount < 0 {
		return &ValidationError{
			Field:   "min_txn_count",
			Message: "must be non-negative",
		}
	}

	if offer.LookbackDays < 0 {
		return &ValidationError{
			Field:   "lookback_days",
			Message: "must be non-negative",
		}
	}

	if offer.LookbackDays > 365 {
		return &ValidationError{
			Field:   "lookback_days",
			Message: "cannot exceed 365 days",
		}
	}

	if offer.StartsAt.IsZero() {
		return &ValidationError{
			Field:   "starts_at",
			Message: "is required",
		}
	}

	if offer.EndsAt.IsZero() {
		return &ValidationError{
			Field:   "ends_at",
			Message: "is required",
		}
	}

	if !offer.StartsAt.Before(offer.EndsAt) {
		return &ValidationError{
			Field:   "starts_at",
			Message: "must be before ends_at",
		}
	}

	// Validate time range is reasonable (not more than 2 years)
	maxDuration := 2 * 365 * 24 * time.Hour
	if offer.EndsAt.Sub(offer.StartsAt) > maxDuration {
		return &ValidationError{
			Field:   "ends_at",
			Message: "offer duration cannot exceed 2 years",
		}
	}

	return nil
}

// ValidateTransaction validates a transaction and returns any validation errors.
func ValidateTransaction(txn models.Transaction) error {
	if err := ValidateUUID(txn.ID, "id"); err != nil {
		return err
	}

	if err := ValidateUUID(txn.UserID, "user_id"); err != nil {
		return err
	}

	if err := ValidateUUID(txn.MerchantID, "merchant_id"); err != nil {
		return err
	}

	if err := validateMCC(txn.MCC); err != nil {
		return err
	}

	if txn.AmountCents < 0 {
		return &ValidationError{
			Field:   "amount_cents",
			Message: "must be non-negative",
		}
	}

	// Reasonable upper limit for transaction amount (1 million dollars)
	maxAmount := int64(100_000_000)
	if txn.AmountCents > maxAmount {
		return &ValidationError{
			Field:   "amount_cents",
			Message: "exceeds maximum allowed amount",
		}
	}

	if txn.ApprovedAt.IsZero() {
		return &ValidationError{
			Field:   "approved_at",
			Message: "is required",
		}
	}

	// Validate timestamp is not too far in the future (allow 1 hour buffer for clock skew)
	maxFutureTime := time.Now().Add(1 * time.Hour)
	if txn.ApprovedAt.After(maxFutureTime) {
		return &ValidationError{
			Field:   "approved_at",
			Message: "cannot be more than 1 hour in the future",
		}
	}

	// Validate timestamp is not too far in the past (10 years)
	maxPastTime := time.Now().AddDate(-10, 0, 0)
	if txn.ApprovedAt.Before(maxPastTime) {
		return &ValidationError{
			Field:   "approved_at",
			Message: "cannot be more than 10 years in the past",
		}
	}

	return nil
}

// SanitizeString removes potentially dangerous characters and trims whitespace.
func SanitizeString(s string) string {
	// Remove null bytes and control characters
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)

	return strings.TrimSpace(s)
}

// ValidateUUID validates that a string is a valid UUID v4.
func ValidateUUID(id, fieldName string) error {
	if id == "" {
		return &ValidationError{
			Field:   fieldName,
			Message: "is required",
		}
	}

	// Sanitize before validation
	id = SanitizeString(id)

	if !uuidRegex.MatchString(strings.ToLower(id)) {
		return &ValidationError{
			Field:   fieldName,
			Message: "must be a valid UUID v4",
		}
	}

	return nil
}

// validateMCC validates that a string is a valid 4-digit MCC code.
func validateMCC(mcc string) error {
	if mcc == "" {
		return &ValidationError{
			Field:   "mcc",
			Message: "is required",
		}
	}

	mcc = SanitizeString(mcc)

	if !mccRegex.MatchString(mcc) {
		return &ValidationError{
			Field:   "mcc",
			Message: "must be a 4-digit numeric code",
		}
	}

	return nil
}

// validateMCCWhitelist validates that all MCC codes in the whitelist are valid.
func validateMCCWhitelist(mccList []string) error {
	if len(mccList) == 0 {
		return nil // Empty whitelist is valid
	}

	if len(mccList) > 100 {
		return &ValidationError{
			Field:   "mcc_whitelist",
			Message: "cannot contain more than 100 MCC codes",
		}
	}

	seen := make(map[string]bool)
	for i, mcc := range mccList {
		if err := validateMCC(mcc); err != nil {
			return &ValidationError{
				Field:   fmt.Sprintf("mcc_whitelist[%d]", i),
				Message: err.Error(),
			}
		}

		// Check for duplicates
		if seen[mcc] {
			return &ValidationError{
				Field:   "mcc_whitelist",
				Message: fmt.Sprintf("duplicate MCC code: %s", mcc),
			}
		}
		seen[mcc] = true
	}

	return nil
}

// ValidateTimeString validates that a string is a valid RFC3339 timestamp.
func ValidateTimeString(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, &ValidationError{
			Field:   "time",
			Message: "is required",
		}
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, &ValidationError{
			Field:   "time",
			Message: "must be a valid RFC3339 timestamp",
		}
	}

	return t, nil
}
