package port

import (
	"context"
	"errors"
	"goyav/internal/core/domain"
	"io"
)

// DocumentService defines the operations for managing documents in the system.
// It provides methods for uploading documents and retrieving their status.
type DocumentService interface {
	// Upload accepts a byte slice representing a document, along with a tag for the document.
	// It returns the ID of the newly uploaded document and any error encountered during the upload process.
	Upload(ctx context.Context, data io.ReadSeeker, size int64, tag string) (ID string, err error)

	// GetDocument retrieves the current status of a document identified by its ID.
	// It returns the document information (if found) and any error encountered during the retrieval process.
	GetDocument(ctx context.Context, ID string) (*domain.Document, error)

	// Ping checks the connectivity or readiness of the service.
	Ping() error

	// Version returns the current version of the DocumentService.
	Version() string

	// Information provides a brief summary or description of the DocumentService.
	// It could include information like the service capabilities, API specifications, etc.
	Information() string
}

var (
	// ErrServiceUploadFailed is returned when uploading a document fails.
	ErrServiceUploadFailed = errors.New("failed to upload data")

	// ErrServiceNoDataToUpload is returned when there is no data available for upload.
	ErrServiceNoDataToUpload = errors.New("no data to upload")

	// ErrServiceGetDocumentFailed is returned when retrieving a document fails.
	ErrServiceGetDocumentFailed = errors.New("failed to retrieve document")

	// ErrUserServiceInvalidID indicates that an invalid ID was provided.
	ErrServiceInvalidID = errors.New("invalid ID provided")
)
