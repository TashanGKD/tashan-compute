package credentials

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{values: make(map[string]string)}
}

func (store *MemoryStore) Save(_ context.Context, label, secret string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	if err := validateSecret(secret); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.values[label] = secret
	return nil
}

func (store *MemoryStore) Load(_ context.Context, label string) (string, bool, error) {
	if err := validateLabel(label); err != nil {
		return "", false, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	value, found := store.values[label]
	return value, found, nil
}

func (store *MemoryStore) Delete(_ context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.values, label)
	return nil
}
