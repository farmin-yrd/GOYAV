package docrepo

import (
	"context"
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"goyav/pkg/helper"
	"maps"
	"sync"
	"time"
)

// MockDocumentRepository is a mock implementation of the DocumentRepository interface.
// It uses an in-memory map to simulate document storage.
type MockDocumentRepository struct {
	documents   map[string]*domain.Document
	documentMux sync.Mutex

	isOnline  bool
	onlineMux sync.Mutex
}

var ErrMockDocumentRepository = errors.New("MockDocumentRepository")

// NewMock creates a new instance of MockDocumentRepository.
func NewMock() *MockDocumentRepository {
	return &MockDocumentRepository{
		documents: make(map[string]*domain.Document),
		isOnline:  true,
	}
}

// Get retrieves a document by its ID. Returns an error if the document does not exist or if a prob
func (m *MockDocumentRepository) Get(ctx context.Context, id string) (*domain.Document, error) {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return nil, err
	}
	m.documentMux.Lock()
	defer m.documentMux.Unlock()
	if doc, exists := m.documents[id]; exists {
		return doc, nil
	}
	return nil, fmt.Errorf("%w: %w: id=%q", ErrMockDocumentRepository, port.ErrDocumentNotFound, id)
}

// Save adds a new document to the repository.
func (m *MockDocumentRepository) Save(ctx context.Context, d *domain.Document) error {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return err
	}
	doc, _ := m.Get(ctx, d.ID)
	if doc != nil {
		return fmt.Errorf("%w: %w: %w: id=%q", ErrMockDocumentRepository, port.ErrSaveDocumentFailed, port.ErrDocumentAlreadyExists, doc.ID)
	}
	m.documentMux.Lock()
	defer m.documentMux.Unlock()
	m.documents[d.ID] = d
	return nil
}

// GetByHash retrieves a document by its hash.
func (m *MockDocumentRepository) GetByHash(ctx context.Context, h string) (*domain.Document, error) {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return nil, err
	}
	if !helper.IsValidHash(h) {
		return nil, fmt.Errorf("%w: invalid hash: %q", ErrMockDocumentRepository, h)
	}
	m.documentMux.Lock()
	defer m.documentMux.Unlock()
	for _, doc := range m.documents {
		if doc.Hash == h {
			return doc, nil
		}
	}
	return nil, fmt.Errorf("%w: %w: hash=%q", ErrMockDocumentRepository, port.ErrDocumentNotFound, h)
}

// Delete removes a document from the repository.
func (m *MockDocumentRepository) Delete(ctx context.Context, id string) error {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return err
	}
	_, err := m.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w: %w", ErrMockDocumentRepository, port.ErrDeleteDocumentFailed, err)
	}
	m.documentMux.Lock()
	defer m.documentMux.Unlock()
	delete(m.documents, id)
	return nil
}

// UpdateStatus updates the analysis status and the analysis date of a document.
func (m *MockDocumentRepository) UpdateStatus(ctx context.Context, id string, status domain.AnalysisStatus, analyzedAt time.Time) error {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return err
	}
	doc, err := m.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w: %w", ErrMockDocumentRepository, port.ErrUpdateStatusFailed, err)
	}
	m.documentMux.Lock()
	defer m.documentMux.Unlock()
	doc.Status = status
	doc.AnalyzedAt = analyzedAt
	return nil
}

// Ping checks the availability of the repository.
func (m *MockDocumentRepository) Ping() error {
	// Simulate a condition that would cause the ping operation to fail.
	if !m.isOnline {
		return fmt.Errorf("%w: %w", ErrMockDocumentRepository, port.ErrDocumentRepositoryUnavailable)
	}
	return nil
}

// Purge removes documents from the repository that have a known antiviral analysis result
// and were created before the specified date.
func (m *MockDocumentRepository) Purge(date time.Time) error {
	if !m.isOnline {
		return fmt.Errorf("%w: document repository is offline", ErrMockDocumentRepository)
	}
	m.documentMux.Lock()
	defer m.documentMux.Unlock()
	maps.DeleteFunc(m.documents, func(k string, v *domain.Document) bool {
		return v.CreatedAt.Before(date) && v.Status != domain.StatusPending
	})
	return nil
}

// Online switches on or off the status of a mock document repository instance.
func (m *MockDocumentRepository) IsOnline(b bool) {
	m.onlineMux.Lock()
	m.isOnline = b
	m.onlineMux.Unlock()
}

func (m *MockDocumentRepository) checkContextAndAvailability(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrMockDocumentRepository, err)
	}
	if !m.isOnline {
		return fmt.Errorf("%w: document repository is Offline", ErrMockDocumentRepository)
	}
	return nil
}
