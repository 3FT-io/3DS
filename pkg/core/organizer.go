package core

import (
	"context"
	"errors"
	"sync"
)

// Error definitions
var (
	ErrCollectionNotFound = errors.New("collection not found")
)

// Organizer manages object relationships and collections
type Organizer struct {
	mu          sync.RWMutex
	collections map[string]*Collection
	referencer  *Referencer
}

// Collection represents a group of related objects
type Collection struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Objects  []string               `json:"objects"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewOrganizer creates a new organizer instance
func NewOrganizer(referencer *Referencer) *Organizer {
	return &Organizer{
		collections: make(map[string]*Collection),
		referencer:  referencer,
	}
}

// CreateCollection creates a new collection
func (o *Organizer) CreateCollection(ctx context.Context, name string) (*Collection, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	collection := &Collection{
		Name:     name,
		Objects:  make([]string, 0),
		Metadata: make(map[string]interface{}),
	}

	o.collections[collection.ID] = collection
	return collection, nil
}

// AddToCollection adds an object to a collection
func (o *Organizer) AddToCollection(ctx context.Context, collectionID, objectID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	collection, ok := o.collections[collectionID]
	if !ok {
		return ErrCollectionNotFound
	}

	collection.Objects = append(collection.Objects, objectID)
	return nil
}

// GetCollection retrieves a collection by ID
func (o *Organizer) GetCollection(ctx context.Context, id string) (*Collection, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	collection, ok := o.collections[id]
	if !ok {
		return nil, ErrCollectionNotFound
	}

	return collection, nil
}
