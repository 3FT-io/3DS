package core

import (
	"context"
	"sync"
)

// Referencer manages references between objects and their dependencies
type Referencer struct {
	mu       sync.RWMutex
	refs     map[string][]string // object ID -> referenced object IDs
	backRefs map[string][]string // object ID -> referencing object IDs
}

// NewReferencer creates a new referencer instance
func NewReferencer() *Referencer {
	return &Referencer{
		refs:     make(map[string][]string),
		backRefs: make(map[string][]string),
	}
}

// AddReference adds a reference from one object to another
func (r *Referencer) AddReference(ctx context.Context, fromID, toID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add forward reference
	r.refs[fromID] = append(r.refs[fromID], toID)

	// Add backward reference
	r.backRefs[toID] = append(r.backRefs[toID], fromID)

	return nil
}

// GetReferences returns all objects referenced by the given object
func (r *Referencer) GetReferences(ctx context.Context, objectID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	refs := make([]string, len(r.refs[objectID]))
	copy(refs, r.refs[objectID])
	return refs, nil
}

// GetBackReferences returns all objects that reference the given object
func (r *Referencer) GetBackReferences(ctx context.Context, objectID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	refs := make([]string, len(r.backRefs[objectID]))
	copy(refs, r.backRefs[objectID])
	return refs, nil
}
