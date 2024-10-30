package core

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type ModelMetadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Format      string    `json:"format"`
	Size        int64     `json:"size"`
	Hash        string    `json:"hash"`
	Chunks      []string  `json:"chunks"`
	CreatedAt   time.Time `json:"created_at"`
	Owner       string    `json:"owner"`
	Permissions []string  `json:"permissions"`
}

type ModelChunk struct {
	ID      string `json:"id"`
	Data    []byte `json:"data"`
	Index   int    `json:"index"`
	ModelID string `json:"model_id"`
	Hash    string `json:"hash"`
}

func (m *ModelMetadata) CalculateHash() string {
	h := sha256.New()
	h.Write([]byte(m.ID + m.Name + m.Format))
	return fmt.Sprintf("%x", h.Sum(nil))
}
