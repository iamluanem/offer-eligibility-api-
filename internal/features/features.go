package features

import (
	"context"
	"sync"
)

type FeatureFlag struct {
	Name        string
	Enabled     bool
	Description string
}

type Manager struct {
	mu     sync.RWMutex
	flags  map[string]*FeatureFlag
	ctx    context.Context
	cancel context.CancelFunc
}

func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		flags:  make(map[string]*FeatureFlag),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (m *Manager) Register(name string, enabled bool, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.flags[name] = &FeatureFlag{
		Name:        name,
		Enabled:     enabled,
		Description: description,
	}
}

func (m *Manager) IsEnabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, exists := m.flags[name]
	if !exists {
		return false
	}

	return flag.Enabled
}

func (m *Manager) Enable(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flag, exists := m.flags[name]; exists {
		flag.Enabled = true
	}
}

func (m *Manager) Disable(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flag, exists := m.flags[name]; exists {
		flag.Enabled = false
	}
}

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

func (m *Manager) Shutdown() {
	m.cancel()
}

const (
	FeatureCacheEnabled        = "cache_enabled"
	FeatureEventHooksEnabled   = "event_hooks_enabled"
	FeatureAdvancedEligibility = "advanced_eligibility"
	FeatureBatchProcessing     = "batch_processing"
)
