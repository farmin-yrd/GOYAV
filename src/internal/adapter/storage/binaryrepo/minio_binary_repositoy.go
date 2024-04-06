package binaryrepo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"goyav/internal/core/port"

	"github.com/minio/minio-go/v7"
)

// MinioBinaryRepository provides a storage backend using Minio.
type MinioBinaryRepository struct {
	client     *minio.Client
	bucketName string
}

var ErrMinioBinaryRepository = errors.New("MinioBinaryRepository")

// NewMinio creates a new instance of MinioByteRepository.
func NewMinio(client *minio.Client, bucketName string) (*MinioBinaryRepository, error) {

	if client == nil {
		return nil, fmt.Errorf("%w: client is nil", ErrMinioBinaryRepository)
	}

	if bucketName == "" {
		return nil, fmt.Errorf("%w: bucket name is empty", ErrMinioBinaryRepository)
	}

	bucketExists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMinioBinaryRepository, err)
	}

	// if the named bucket doesn't exist, create it.
	if !bucketExists {
		if err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMinioBinaryRepository, err)
		}
		slog.Debug("a new bucket is created")
	}

	return &MinioBinaryRepository{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// Save saves an object into the Minio bucket
func (m *MinioBinaryRepository) Save(ctx context.Context, data io.Reader, size int64, ID string) error {
	_, err := m.client.PutObject(ctx, m.bucketName, ID, io.LimitReader(data, size), size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrMinioBinaryRepository, port.ErrSaveDataFailed, err)
	}

	return nil
}

// Delete removes an object from the Minio bucket identified by ID. Returns an error if the object is not found.
func (m MinioBinaryRepository) Delete(ctx context.Context, ID string) error {
	if err := m.exists(ctx, ID); err != nil {
		return fmt.Errorf("%w: %w: %v", ErrMinioBinaryRepository, port.ErrDeleteDataFailed, err)
	}
	err := m.client.RemoveObject(ctx, m.bucketName, ID, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrMinioBinaryRepository, port.ErrDeleteDataFailed, err)
	}
	return nil
}

// Get returns an object from the Minio bucket identified by ID. Returns error if the object does not exist.
func (m MinioBinaryRepository) Get(ctx context.Context, ID string) (io.ReadCloser, error) {
	if err := m.exists(ctx, ID); err != nil {
		return nil, fmt.Errorf("%w: %w: %v", ErrMinioBinaryRepository, port.ErrGetDataFailed, err)
	}
	o, err := m.client.GetObject(ctx, m.bucketName, ID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %w: %v", ErrMinioBinaryRepository, port.ErrGetDataFailed, err)
	}
	return o, nil
}

// Ping checks Minio service availability with a 5-second timeout.
func (m MinioBinaryRepository) Ping() error {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := m.client.ListBuckets(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %w: timeout after %s", ErrMinioBinaryRepository, port.ErrBinaryRepositoryUnavailable, timeout)
		}
		return fmt.Errorf("%w: %w: %v", ErrMinioBinaryRepository, port.ErrBinaryRepositoryUnavailable, err)
	}

	return nil
}

// exists checks if an object with the given ID exists in the repository.
func (m MinioBinaryRepository) exists(ctx context.Context, ID string) error {
	if _, err := m.client.StatObject(ctx, m.bucketName, ID, minio.StatObjectOptions{}); err != nil {
		return fmt.Errorf("error while searching for ID = %q: %w", ID, err)
	}
	return nil
}
