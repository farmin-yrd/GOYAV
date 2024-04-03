package port

import (
	"context"
	"errors"
	"goyav/internal/core/domain"
	"io"
)

// AntivirusAnalyzer defines the interface for performing antivirus analysis on documents.
// It provides a method to analyze the byte content of a document for potential threats.
type AntivirusAnalyzer interface {
	// Analyze performs antivirus analysis on the byte content of a document.
	// It accepts a context.Context to handle cancellations and timeouts.
	// This method returns an AnalysisStatus (e.g., StatusClean, StatusInfected)
	// and an error if the analysis cannot be completed.
	Analyze(ctx context.Context, data io.Reader) (domain.AnalysisStatus, error)

	// TimeoutValue returns the timeout value for the antivirus analyzer in seconds.
	TimeoutValue() uint64

	// Ping checks the connectivity or readiness of the antivirus service.
	// It returns an error if the service is not reachable or not ready.
	Ping() error
}

var (
	EICAR = []byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`)

	// ErrAntivirusAnalysisFailed is returned when the analysis cannot be performed due to an error in the process.
	ErrAntivirusAnalysisFailed = errors.New("antivirus analysis failed")

	// ErrAntivirusServiceUnavailable is returned when the antivirus service is not reachable or not ready.
	ErrAntivirusAnalyserUnavailable = errors.New("antivirus service is unavailable")
)
