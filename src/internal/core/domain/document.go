package domain

import (
	"time"
)

// AnalysisStatus represents the status of document analysis.
type AnalysisStatus int

const (
	// StatusPending indicates that the document analysis is pending.
	StatusPending AnalysisStatus = iota

	// StatusInfected indicates that the document is infected by a virus.
	StatusInfected

	// StatusClean indicates that the document is clean (not infected).
	StatusClean
)

// Document represents a document with its attributes.
type Document struct {
	ID         string         `json:"id"`
	Hash       string         `json:"hash"`
	Tag        string         `json:"tag"`
	Status     AnalysisStatus `json:"status"`
	AnalyzedAt time.Time      `json:"analyzed_at"`
	CreatedAt  time.Time      `json:"created_at"`
}

// NewDocument creates a new Document instance with the provided ID, hash and tag.
func NewDocument(id, hash, tag string) *Document {
	return &Document{
		ID:        id,
		Hash:      hash,
		Tag:       tag,
		CreatedAt: time.Now(),
		Status:    StatusPending,
	}
}
