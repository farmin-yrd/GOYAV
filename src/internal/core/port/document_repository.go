// Package port defines interfaces for the document domain operations.
package port

import (
	"context"
	"errors"
	"goyav/internal/core/domain"
	"time"
)

// DocumentRepository defines operations for managing documents in a repository.
type DocumentRepository interface {
	// Save adds a new document to the repository and returns an error if the document already exists or
	// if there is an issue during the save operation.
	Save(ctx context.Context, doc *domain.Document) error

	// Get retrieves a document by its ID and returns an error if not found or if there is an issue with the ID.
	Get(ctx context.Context, id string) (*domain.Document, error)

	// GetByHash retrieves a document by its hash and returns an error if not found or if there is an issue with the hash.
	GetByHash(ctx context.Context, hash string) (*domain.Document, error)

	// Delete removes a document from the repository by its ID and returns an error if not found or during deletion.
	Delete(ctx context.Context, id string) error

	// UpdateStatus updates a document's analysis status and date, returning an error for nonexistent documents,
	// invalid status, or update issues.
	UpdateStatus(ctx context.Context, id string, status domain.AnalysisStatus, analyzedAt time.Time) error

	// Ping checks the repository's availability or health status.
	Ping() error

	// Purge removes documents from the repository that have a known antiviral analysis result
	// and were created before the specified date.
	Purge(date time.Time) error
}

var (
	// ErrDocumentAlreadyExists indicates an attempt to save a document that already exists in the repository.
	ErrDocumentAlreadyExists = errors.New("document already exists")

	// ErrDocumentNotFound indicates that the specified document could not be found in the repository.
	ErrDocumentNotFound = errors.New("document not found")

	// ErrGetDocumentFailed indicates a failure to get a document from the repository,
	// possibly due to database or connectivity issues.
	ErrGetDocumentFailed = errors.New("failed to get document")

	// ErrUpdateStatusFailed indicates a failure in updating the status of a document,
	// possibly due to a nonexistent document or database issues.
	ErrUpdateStatusFailed = errors.New("failed to update document status")

	// ErrSaveDocumentFailed indicates a failure to save a new document to the repository,
	// possibly due to database or connectivity issues.
	ErrSaveDocumentFailed = errors.New("failed to save the document")

	// ErrDeleteDocumentFailed indicates a failure to delete a document from the repository,
	// possibly due to the document not existing or database issues.
	ErrDeleteDocumentFailed = errors.New("failed to delete the document")

	// ErrDocumentRepositoryUnavailable indicates that the document repository is not accessible,
	// possibly due to database downtime or network issues.
	ErrDocumentRepositoryUnavailable = errors.New("document repository is unavailable")

	// ErrDocumentRepositoryPurgeFaild indicates a failure in the purge operation of the document repository.
	ErrDocumentRepositoryPurgeFailed = errors.New("failed to purge the document repository")
)
