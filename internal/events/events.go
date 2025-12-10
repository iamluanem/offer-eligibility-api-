package events

import (
	"context"
	"sync"
	"time"

	"offer-eligibility-api/internal/models"
)

// EventType represents the type of event.
type EventType string

const (
	// EventOfferCreated is emitted when an offer is created or updated
	EventOfferCreated EventType = "offer.created"
	// EventTransactionCreated is emitted when transactions are created
	EventTransactionCreated EventType = "transaction.created"
	// EventEligibilityChecked is emitted when eligibility is checked for a user
	EventEligibilityChecked EventType = "eligibility.checked"
)

// Event represents an event in the system.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// OfferCreatedData contains data for offer created events.
type OfferCreatedData struct {
	Offer models.Offer
}

// TransactionCreatedData contains data for transaction created events.
type TransactionCreatedData struct {
	Transactions []models.Transaction
	Count        int
}

// EligibilityCheckedData contains data for eligibility checked events.
type EligibilityCheckedData struct {
	UserID         string
	EligibleOffers []models.EligibleOffer
	CheckedAt      time.Time
}

// Handler is a function that handles events.
type Handler func(ctx context.Context, event Event) error

// Manager manages event handlers and event publishing.
type Manager struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
	enabled  bool
}

// NewManager creates a new event manager.
func NewManager(enabled bool) *Manager {
	return &Manager{
		handlers: make(map[EventType][]Handler),
		enabled:  enabled,
	}
}

// Subscribe subscribes a handler to a specific event type.
func (m *Manager) Subscribe(eventType EventType, handler Handler) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

// Publish publishes an event to all subscribed handlers.
func (m *Manager) Publish(ctx context.Context, eventType EventType, data interface{}) {
	if !m.enabled {
		return
	}

	m.mu.RLock()
	handlers := m.handlers[eventType]
	m.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Execute handlers asynchronously to avoid blocking
	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				// In production, you might want to log this or send to error tracking
				_ = err
			}
		}(handler)
	}
}

// PublishOfferCreated publishes an offer created event.
func (m *Manager) PublishOfferCreated(ctx context.Context, offer models.Offer) {
	m.Publish(ctx, EventOfferCreated, OfferCreatedData{Offer: offer})
}

// PublishTransactionCreated publishes a transaction created event.
func (m *Manager) PublishTransactionCreated(ctx context.Context, transactions []models.Transaction, count int) {
	m.Publish(ctx, EventTransactionCreated, TransactionCreatedData{
		Transactions: transactions,
		Count:        count,
	})
}

// PublishEligibilityChecked publishes an eligibility checked event.
func (m *Manager) PublishEligibilityChecked(ctx context.Context, userID string, eligibleOffers []models.EligibleOffer) {
	m.Publish(ctx, EventEligibilityChecked, EligibilityCheckedData{
		UserID:         userID,
		EligibleOffers: eligibleOffers,
		CheckedAt:      time.Now(),
	})
}

// Shutdown shuts down the event manager.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = false
	m.handlers = make(map[EventType][]Handler)
}

