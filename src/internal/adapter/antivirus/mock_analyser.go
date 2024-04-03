package antivirus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"io"
	"time"
)

// MockAntivirusAnalyzer is a mock implementation of the AntivirusAnalyzer interface.
// It uses the EICAR test byte slice to simulate virus detection.
type MockAntivirusAnalyzer struct {
	isOnline bool          // Indicates whether the mock analyzer is "online" or "offline"
	Timeout  time.Duration // Timeout in seconds
}

var ErrMockAntivirusAnalyzer = errors.New("MockAntivirusAnalyzer")

// NewMock creates a new instance of MockAntivirusAnalyzer.
func NewMock() *MockAntivirusAnalyzer {
	return &MockAntivirusAnalyzer{
		isOnline: true,
		Timeout:  60 * time.Second,
	}
}

// Analyze performs a mock antivirus analysis on the byte content of a document.
func (m *MockAntivirusAnalyzer) Analyze(ctx context.Context, r io.Reader) (domain.AnalysisStatus, error) {
	if err := m.checkContextAndAvailability(ctx); err != nil {
		return domain.StatusPending, err
	}

	// Simulate analysis duration
	time.Sleep(time.Second)

	var status domain.AnalysisStatus
	b, err := io.ReadAll(r)
	if err != nil {
		return domain.StatusPending, fmt.Errorf("%w: %w: %v", ErrMockAntivirusAnalyzer, port.ErrAntivirusAnalysisFailed, err)
	}
	if bytes.Contains(b, port.EICAR) {
		status = domain.StatusInfected
	} else {
		status = domain.StatusClean
	}
	return status, nil
}

// Ping simulates a connectivity check to the antivirus service.
func (m *MockAntivirusAnalyzer) Ping() error {
	if !m.isOnline {
		return fmt.Errorf("%w: %w", ErrMockAntivirusAnalyzer, port.ErrAntivirusAnalyserUnavailable)
	}
	return nil
}

// TimeoutValue returns the timeout value of the analyzer.
func (m *MockAntivirusAnalyzer) TimeoutValue() uint64 {
	return uint64(m.Timeout.Seconds())
}

// Online switches on or off the status of a mock analyzer instance.
func (m *MockAntivirusAnalyzer) IsOnline(b bool) {
	m.isOnline = b
}

func (m *MockAntivirusAnalyzer) checkContextAndAvailability(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrMockAntivirusAnalyzer, err)
	}
	if !m.isOnline {
		return fmt.Errorf("%w:  antivirus analyze is Offline", ErrMockAntivirusAnalyzer)
	}
	return nil
}
