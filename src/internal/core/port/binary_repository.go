package port

import (
	"context"
	"errors"
	"io"
)

// BinaryRepository defines the interface for operations related to managing the binary data of documents.
// This interface abstracts the underlying storage mechanism, which could be a file system or
// an object storage system like AWS S3, Azure Blob Storage, or MinIO.
type BinaryRepository interface {
	// Save stores the binary data of a document, identified by a unique ID, into the storage system.
	// The function takes a context to manage timeouts and cancellation, a reader for the data,
	// the size of the data, and the document's ID.
	Save(ctx context.Context, data io.Reader, size int64, ID string) error

	// Get retrieves the binary data of a document identified by the given ID.
	// It returns an io.ReadCloser to read the document's data and an error, if any occurred.
	Get(ctx context.Context, ID string) (io.ReadCloser, error)

	// Delete removes the binary data associated with the given document ID from the storage system.
	Delete(ctx context.Context, ID string) error

	// Ping checks the availability or health of the storage system. It is used to verify
	// if the storage system is accessible and functioning correctly.
	Ping() error
}

var (
	// ErrSaveDataFailed is returned when the Save operation fails.
	ErrSaveDataFailed = errors.New("failed to save the document's bytes data")

	// ErrGetDataFailed is returned when the Get operation fails.
	ErrGetDataFailed = errors.New("failed to get the document's bytes data")

	// ErrDeleteDataFailed is returned when the Delete operation fails.
	ErrDeleteDataFailed = errors.New("failed to delete the document's bytes data")

	// ErrBinaryRepositoryUnavailable is returned when the Ping operation fails to reach the byte repository.
	ErrBinaryRepositoryUnavailable = errors.New("binary repository is unavailable")
)
