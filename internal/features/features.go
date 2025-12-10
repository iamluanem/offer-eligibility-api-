package features

import (
	"context"
	"sync"
)

// FeatureFlag represents a feature flag configuration.
type FeatureFlag struct {
	Name        string
	Enabled     bool
	Description string
}

// Manager manages feature flags.
type Manager struct {
	mu     sync.RWMutex
	flags  map[string]*FeatureFlag
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new feature flag manager.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		flags:  make(map[string]*FeatureFlag),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Register registers a new feature flag.
func (m *Manager) Register(name string, enabled bool, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.flags[name] = &FeatureFlag{
		Name:        name,
		Enabled:     enabled,
		Description: description,
	}
}

// IsEnabled checks if a feature flag is enabled.
func (m *Manager) IsEnabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, exists := m.flags[name]
	if !exists {
		return false // Default to disabled if flag doesn't exist
	}

	return flag.Enabled
}

// Enable enables a feature flag.
func (m *Manager) Enable(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flag, exists := m.flags[name]; exists {
		flag.Enabled = true
	}
}

// Disable disables a feature flag.
func (m *Manager) Disable(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flag, exists := m.flags[name]; exists {
		flag.Enabled = false
	}
}

// GetAll returns all feature flags.
func (m *Manager) GetAll() map[string]*FeatureFlag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*FeatureFlag)
	for k, v := range m.flags {
		result[k] = &FeatureFlag{
			Name:        v.Name,
			Enabled:     v.Enabled,
			Description: v.Description,
		}
	}
	return result
}

// Shutdown shuts down the feature flag manager.
func (m *Manager) Shutdown() {
	m.cancel()
}

// Predefined feature flag names
const (
	// FeatureCacheEnabled enables/disables caching layer
	FeatureCacheEnabled = "cache_enabled"
	// FeatureEventHooksEnabled enables/disables event-driven hooks
	FeatureEventHooksEnabled = "event_hooks_enabled"
	// FeatureAdvancedEligibility enables advanced eligibility calculations
	FeatureAdvancedEligibility = "advanced_eligibility"
	// FeatureBatchProcessing enables batch transaction processing optimizations
	FeatureBatchProcessing = "batch_processing"
)
