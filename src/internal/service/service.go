package service

import (
	"context"
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"goyav/pkg/helper"
	"io"
	"log/slog"
	"time"
)

// Service manages file uploads and antivirus analysis operations.
type Service struct {
	BinayRepository    port.BinaryRepository
	DocumentRepository port.DocumentRepository
	AvAnalyzer         port.AntivirusAnalyzer

	// semaphore is used to control concurrent access to resources.
	semaphore chan struct{}

	// version is the current version of the service
	version string

	// information about the service
	information string

	// resultTimeToLive specifies the duration for which analysis results are retained.
	resultTimeToLive time.Duration
}

const (
	// DefaultSemaphoreCapacity represents the default number of parallel goroutines
	// that the server can run
	DefaultSemaphoreCapacity = uint64(128)
)

var (
	// AntivirusRetryWaitTimes represents the time intervals in seconds before retrying
	// an antivirus analysis after a connection failure.
	AntivirusRetryWaitTimes = []int64{5, 10, 15, 25, 40, 65}

	// ErrNilDependency is an error that occurs when a required dependency is nil
	ErrNilDependency = errors.New("Service: nil dependency")
)

// New creates a new Service instance with the specified dependencies, including binary repository,
// document repository, antivirus analyzer, and additional service information like version, info,
// result time-to-live, auto-purge flag, and semaphore capacity. It validates the dependencies and initializes
// the Service with default or specified settings. If result time-to-if is strcitly positive, it starts
// the purge process as a separate goroutine. Returns an error if dependencies are missing or if initial pinging of
// repositories and analyzer fails.
func New(binaryRepo port.BinaryRepository, docRepo port.DocumentRepository, avAnalyzer port.AntivirusAnalyzer, version, info string, resTTL time.Duration, semaphoreCapacity uint64) (*Service, error) {
	if binaryRepo == nil || docRepo == nil || avAnalyzer == nil {
		return nil, fmt.Errorf("%w: missing repositories or analyzer", ErrNilDependency)
	}

	if err := ping(binaryRepo, docRepo, avAnalyzer); err != nil {
		return nil, fmt.Errorf("service: unable to create: %w", err)
	}

	autoPurge := (resTTL > 0)

	capacity := max(semaphoreCapacity, DefaultSemaphoreCapacity)

	service := &Service{
		BinayRepository:    binaryRepo,
		DocumentRepository: docRepo,
		AvAnalyzer:         avAnalyzer,
		semaphore:          make(chan struct{}, capacity),
		version:            version,
		information:        info,
		resultTimeToLive:   resTTL,
	}

	if autoPurge {
		go service.autoPurge()
	}

	return service, nil
}

// Version returns the current version of the service.
func (s *Service) Version() string {
	return s.version
}

// Information returns the information about the service
func (s *Service) Information() string {
	return s.information
}

// Upload handles the uploading of a document to the service. It computes a hash of the document,
// sanitizes the provided tag, checks for the existence of a document with the same hash,
// and either returns the ID of the existing document or saves a new one and triggers antivirus analysis.
func (s *Service) Upload(ctx context.Context, data io.Reader, size int64, tag string) (ID string, err error) {
	// Sanitize the tag.
	tag = helper.Sanitize(tag)

	// new CryptoWriter for generating hash and ID
	cw := helper.NewCryptoWriter()
	data = io.TeeReader(io.LimitReader(data, size), cw)

	// Calculate the hash of the document and Generate its ID
	hash, ID, err := cw.GenerateHashAndID(tag)
	if err != nil {
		return "", fmt.Errorf("service: failed to calculate the hash or creating a document ID : %w", err)
	}

	// Check if a document with the same hash already exists.
	existingDoc, _ := s.DocumentRepository.GetByHash(ctx, hash)
	if existingDoc != nil {
		// Return existing document's ID if it has the same tag.
		if existingDoc.Tag == tag {
			return existingDoc.ID, port.ErrDocumentAlreadyExists
		}

		// Otherwise save the document with a new ID if it's not pending analysis.
		if existingDoc.Status != domain.StatusPending {
			err = s.DocumentRepository.Save(ctx, &domain.Document{
				ID:         ID,
				Hash:       hash,
				Tag:        tag,
				Status:     existingDoc.Status,
				AnalyzedAt: existingDoc.AnalyzedAt,
				CreatedAt:  time.Now(),
			})
			if err != nil {
				return "", fmt.Errorf("service: %w: %w", port.ErrServiceUploadFailed, err)
			}
			return ID, port.ErrDocumentAlreadyExists
		}
	}

	// Save the binary data.
	if err = s.BinayRepository.Save(ctx, data, size, ID); err != nil {
		return "", fmt.Errorf("service: %w: %w: id=%v", port.ErrServiceUploadFailed, err, ID)
	}

	// Create and save a new document.
	newDoc := domain.NewDocument(ID, hash, tag)
	if err = s.DocumentRepository.Save(ctx, newDoc); err != nil {
		return "", fmt.Errorf("service: %w: %w", port.ErrServiceUploadFailed, err)
	}

	// Trigger an asynchronous antivirus analysis.
	go s.asyncAnalyze(ID)

	return ID, nil
}

// GetDocument retrieves the current status of a document by its ID.
func (s *Service) GetDocument(ctx context.Context, ID string) (*domain.Document, error) {
	if !helper.IsValidID(ID) {
		return nil, fmt.Errorf("service: %w: the provided ID is not valid", port.ErrServiceInvalidID)
	}
	document, err := s.DocumentRepository.Get(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: id=%s", port.ErrServiceGetDocumentFailed, err, ID)
	}
	return document, nil
}

func (s *Service) Ping() error {
	err := ping(s.BinayRepository, s.DocumentRepository, s.AvAnalyzer)
	if err != nil {
		slog.Error("service - ping failed", "error", err)
	}
	return err
}

const asyncAnalyseErrorMsg = "service - async analysis error"

// asyncAnalyze performs the analysis of the data asynchronously with retry attempts
func (s *Service) asyncAnalyze(ID string) {
	s.semaphore <- struct{}{}
	go func() {
		defer func() {
			<-s.semaphore
		}()

		ctx := context.Background()

		// Retrieve and defer close the data stream
		r, err := s.BinayRepository.Get(ctx, ID)
		if err != nil {
			slog.Error(asyncAnalyseErrorMsg, "error", err, "ID", ID)
			return
		}
		defer r.Close()

		// Attempt to analyze with retries
		if err := s.attemptAnalysis(ctx, r, ID); err != nil {
			slog.Error(asyncAnalyseErrorMsg, "error", err, "ID", ID)
		}
		slog.Debug("analyse completed", "ID", ID)

	}()
}

// attemptAnalysis tries to analyze the data with retries.
func (s *Service) attemptAnalysis(ctx context.Context, r io.Reader, ID string) error {
	var status domain.AnalysisStatus
	for _, v := range AntivirusRetryWaitTimes {
		var err error
		if status, err = s.AvAnalyzer.Analyze(ctx, r); err == nil {
			if err = s.DocumentRepository.UpdateStatus(ctx, ID, status, time.Now()); err == nil {
				return s.BinayRepository.Delete(ctx, ID)
			}
			return err
		}
		time.Sleep(time.Second * time.Duration(v))
	}
	return fmt.Errorf("analysis failed after %d attempts", len(AntivirusRetryWaitTimes))
}

func ping(b port.BinaryRepository, d port.DocumentRepository, a port.AntivirusAnalyzer) error {
	return errors.Join(b.Ping(), d.Ping(), a.Ping())
}

// autoPurge periodically purges old documents from the document repository.
// It runs indefinitely, triggering a purge operation at intervals defined by documentTimeToLive.
func (s *Service) autoPurge() {
	ticker := time.NewTicker(s.resultTimeToLive)
	defer ticker.Stop()

	for range ticker.C {
		purgeTime := time.Now().Add(-s.resultTimeToLive)
		if err := s.DocumentRepository.Purge(purgeTime); err != nil {
			slog.Error("service - auto_purge failed", "error", err)
		}
		slog.Debug("service - auto-purge done")
	}
}
