package antivirus

import (
	"context"
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"io"
	"time"

	"github.com/lyimmi/go-clamd"
)

// ClamavAnalyser is an implementation of the AntivirusAnalyzer interface.
type ClamavAnalyser struct {
	Analyser *clamd.Clamd  // Analyser is the ClamAV scanner instance.
	Timeout  time.Duration // Timeout is the timeout value in seconds for operations.
}

var ErrClamavAntiVirusAnalyser = errors.New("ClamavAntiVirusAnalyser")

// NewClamav creates a new instance of ClamavAntiVirusAnalyser.
func NewClamav(host string, port uint64, timeout uint64) (*ClamavAnalyser, error) {
	if timeout == 0 {
		return nil, fmt.Errorf("%w: timeout value must be a strictly positive number. given value=%v", ErrClamavAntiVirusAnalyser, timeout)
	}

	return &ClamavAnalyser{
		Analyser: clamd.NewClamd(
			clamd.WithTCP(host, int(port)),
		),
		Timeout: time.Duration(timeout) * time.Second,
	}, nil
}

// Analyze performs antivirus analysis on the provided binary data.
func (a *ClamavAnalyser) Analyze(ctx context.Context, data io.Reader) (domain.AnalysisStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, a.Timeout)
	defer cancel()
	clean, err := a.Analyser.ScanStream(ctx, data)
	if err != nil {
		if errors.Is(err, clamd.ErrEICARFound) {
			return domain.StatusInfected, nil
		}
		return domain.StatusPending, fmt.Errorf("%w: %w: %v", ErrClamavAntiVirusAnalyser, port.ErrAntivirusAnalysisFailed, err)
	}
	if !clean {
		return domain.StatusInfected, nil
	}
	return domain.StatusClean, nil
}

// TimeoutValue returns the timeout value.
func (a *ClamavAnalyser) TimeoutValue() uint64 {
	return uint64(a.Timeout.Seconds())
}

// Ping checks the connectivity or readiness of the antivirus service.
func (a *ClamavAnalyser) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), a.Timeout)
	defer cancel()
	v, err := a.Analyser.Ping(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrClamavAntiVirusAnalyser, port.ErrAntivirusAnalyserUnavailable, err)
	}
	if !v {
		return fmt.Errorf("%w: %w", ErrClamavAntiVirusAnalyser, port.ErrAntivirusAnalyserUnavailable)
	}
	return nil
}
