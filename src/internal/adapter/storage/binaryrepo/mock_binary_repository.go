package binaryrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"goyav/internal/core/port"
	"goyav/pkg/helper"
	"io"
)

// MockBinaryRepository is a mock implementation of the ByteRepository interface.
// It simulates the behavior of a real repository for testing purposes.
type MockBinaryRepository struct {
	// simulatedStorage simulates a storage system using a map.
	simulatedStorage map[string][]byte
	isOnline         bool
}

// NewMock creates a new instance of MockByteRepository.
func NewMock() *MockBinaryRepository {
	return &MockBinaryRepository{
		simulatedStorage: make(map[string][]byte),
		isOnline:         true,
	}
}

var ErrMockBinaryRepository = errors.New("MockBinaryRepository")

// Save simulates the saving of document's byte data.
// It returns ErrSaveFailed error with additional context if the operation fails.
func (m *MockBinaryRepository) Save(ctx context.Context, data io.Reader, size int64, documentID string) error {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return err
	}

	if !helper.IsValidID(documentID) {
		return fmt.Errorf("%w: %w: invalide id: %q", ErrMockBinaryRepository, port.ErrSaveDataFailed, documentID)
	}

	data = io.LimitReader(data, size)
	b, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("%w: %w: reading data failed: %v", ErrMockBinaryRepository, port.ErrSaveDataFailed, err)
	}
	// Simulate successful save operation.
	m.simulatedStorage[documentID] = b
	return nil
}

// Delete simulates the deletion of document's byte data.
// It returns ErrDeleteFailed error with additional context if the operation fails.
func (m *MockBinaryRepository) Delete(ctx context.Context, documentID string) error {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return err
	}
	if _, exists := m.simulatedStorage[documentID]; !exists {
		return fmt.Errorf("%w: %w : id not found : id=%q", ErrMockBinaryRepository, port.ErrDeleteDataFailed, documentID)
	}

	// Simulate successful delete operation.
	delete(m.simulatedStorage, documentID)
	return nil
}

func (m *MockBinaryRepository) Get(ctx context.Context, ID string) (io.ReadCloser, error) {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return nil, err
	}
	b, exists := m.simulatedStorage[ID]
	if !exists {
		return nil, fmt.Errorf("%w: %w : id not found", ErrMockBinaryRepository, port.ErrGetDataFailed)
	}
	return io.NopCloser(bytes.NewBuffer(b)), nil
}

// Ping simulates a check on the storage system.
// It returns ErrPingByteRepositoryFailed if the simulated ping fails.
func (m *MockBinaryRepository) Ping() error {
	// Simulate a condition that would cause the ping operation to fail.
	if !m.isOnline {
		return fmt.Errorf("%w: %w", ErrMockBinaryRepository, port.ErrBinaryRepositoryUnavailable)
	}

	return nil
}

// Online switches on or off the status of a mock binary repository instance.
func (m *MockBinaryRepository) IsOnline(b bool) {
	m.isOnline = b
}

func (m *MockBinaryRepository) checkContextAndAvailability(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrMockBinaryRepository, err)
	}
	if !m.isOnline {
		return fmt.Errorf("%w: document repository is Offline", ErrMockBinaryRepository)
	}
	return nil
}
