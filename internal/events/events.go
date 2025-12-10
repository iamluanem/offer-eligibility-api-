package events

import (
	"context"
	"sync"
	"time"

	"offer-eligibility-api/internal/models"
)

type EventType string

const (
	EventOfferCreated       EventType = "offer.created"
	EventTransactionCreated EventType = "transaction.created"
	EventEligibilityChecked EventType = "eligibility.checked"
)

type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

type OfferCreatedData struct {
	Offer models.Offer
}

type TransactionCreatedData struct {
	Transactions []models.Transaction
	Count        int
}

type EligibilityCheckedData struct {
	UserID         string
	EligibleOffers []models.EligibleOffer
	CheckedAt      time.Time
}

type Handler func(ctx context.Context, event Event) error

type Manager struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
	enabled  bool
}

func NewManager(enabled bool) *Manager {
	return &Manager{
		handlers: make(map[EventType][]Handler),
		enabled:  enabled,
	}
}

func (m *Manager) Subscribe(eventType EventType, handler Handler) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

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

	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				_ = err
			}
		}(handler)
	}
}

func (m *Manager) PublishOfferCreated(ctx context.Context, offer models.Offer) {
	m.Publish(ctx, EventOfferCreated, OfferCreatedData{Offer: offer})
}

func (m *Manager) PublishTransactionCreated(ctx context.Context, transactions []models.Transaction, count int) {
	m.Publish(ctx, EventTransactionCreated, TransactionCreatedData{
		Transactions: transactions,
		Count:        count,
	})
}

func (m *Manager) PublishEligibilityChecked(ctx context.Context, userID string, eligibleOffers []models.EligibleOffer) {
	m.Publish(ctx, EventEligibilityChecked, EligibilityCheckedData{
		UserID:         userID,
		EligibleOffers: eligibleOffers,
		CheckedAt:      time.Now(),
	})
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = false
	m.handlers = make(map[EventType][]Handler)
}
