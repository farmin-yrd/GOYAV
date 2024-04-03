package docrepo

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

type PostgresDocumentRepository struct {
	db *sql.DB
}

var ErrPostgresDocumentRepository = errors.New("PostgresDocumentRepository")

func NewPotgres(db *sql.DB) (*PostgresDocumentRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("%w : required sql.DB, got nil", ErrPostgresDocumentRepository)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresDocumentRepository, err)
	}
	if err := SetupDocumentTable(db); err != nil {
		return nil, fmt.Errorf("%w: failed to create document table: %v", ErrPostgresDocumentRepository, err)
	}
	slog.Info("document repository created")
	return &PostgresDocumentRepository{db: db}, nil
}

// Save adds a new document to the repository and returns an error if the document already exists or
// if there is an issue during the save operation.
func (r PostgresDocumentRepository) Save(ctx context.Context, doc *domain.Document) error {
	q := "INSERT INTO documents (document_id, hash, tag, status, analyzed_at, created_at) VALUES ($1, $2, $3, $4, $5, $6)"
	_, err := r.db.ExecContext(ctx, q, doc.ID, doc.Hash, doc.Tag, doc.Status, doc.AnalyzedAt, doc.CreatedAt)
	if err != nil {
		return fmt.Errorf("%w: %w: %v: document=%#v", ErrPostgresDocumentRepository, port.ErrSaveDocumentFailed, err, doc)
	}
	return nil
}

// Get retrieves a document by its ID and returns an error if not found or if there is an issue with the ID.
func (r PostgresDocumentRepository) Get(ctx context.Context, ID string) (*domain.Document, error) {
	q := "SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE document_id = $1"
	doc := new(domain.Document)
	err := r.db.QueryRowContext(ctx, q, ID).Scan(
		&doc.ID,
		&doc.Hash,
		&doc.Tag,
		&doc.Status,
		&doc.AnalyzedAt,
		&doc.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrDocumentNotFound, err)
		}

		return nil, fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrGetDocumentFailed, err)
	}
	return doc, nil
}

// GetByHash retrieves a document by its hash and returns an error if not found or if there is an issue with the hash.
func (r PostgresDocumentRepository) GetByHash(ctx context.Context, hash string) (*domain.Document, error) {
	q := "SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE hash = $1"
	doc := new(domain.Document)
	err := r.db.QueryRowContext(ctx, q, hash).Scan(
		&doc.ID,
		&doc.Hash,
		&doc.Tag,
		&doc.Status,
		&doc.AnalyzedAt,
		&doc.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w.GetByHash: %w", ErrPostgresDocumentRepository, port.ErrDocumentNotFound)
		}

		return nil, fmt.Errorf("%w.GetByHash: %w: %v", ErrPostgresDocumentRepository, port.ErrGetDocumentFailed, err)
	}
	return doc, nil
}

// Delete removes a document from the repository by its ID and returns an error if not found or during deletion.
func (r PostgresDocumentRepository) Delete(ctx context.Context, ID string) error {
	q := "DELETE FROM documents WHERE document_id = $1"
	res, err := r.db.ExecContext(ctx, q, ID)
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrDeleteDocumentFailed, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrDeleteDocumentFailed, err)
	}

	if n == 0 {
		return fmt.Errorf("%w: %w: no document found with ID %v", ErrPostgresDocumentRepository, port.ErrDeleteDocumentFailed, ID)
	}

	return nil
}

// UpdateStatus updates a document's analysis status and date, returning an error for nonexistent documents,
// invalid status, or update issues.
func (r PostgresDocumentRepository) UpdateStatus(ctx context.Context, ID string, status domain.AnalysisStatus, analyzedAt time.Time) error {
	q := "UPDATE documents SET status = $1, analyzed_at = $2 WHERE document_id = $3"
	res, err := r.db.ExecContext(ctx, q, status, time.Now(), ID)
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrUpdateStatusFailed, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrUpdateStatusFailed, err)
	}

	if n == 0 {
		return fmt.Errorf("%w: %w: no document found with ID %v", ErrPostgresDocumentRepository, port.ErrUpdateStatusFailed, ID)
	}

	return nil
}

// Ping checks the repository's availability or health status.
func (r PostgresDocumentRepository) Ping() error {
	if err := r.db.Ping(); err != nil {
		return fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrDocumentRepositoryUnavailable, err)
	}
	return nil
}

// Purge removes documents from the repository that were created before the specified date
// and have a status different from pending status (value = 0).
func (r PostgresDocumentRepository) Purge(date time.Time) error {
	q := "DELETE FROM documents WHERE created_at < $1 AND status != $2"
	if _, err := r.db.Exec(q, date, domain.StatusPending); err != nil {
		return fmt.Errorf("%w: %w: %v", ErrPostgresDocumentRepository, port.ErrDocumentRepositoryPurgeFailed, err)
	}
	return nil
}

//go:embed document_table.sql
var createTableQuery string

// SetupDocumentTable sets up the 'documents' table in the database.
// It executes SQL commands to create the table, its indices, and constraints,
// ensuring that the table is properly initialized for use.
func SetupDocumentTable(db *sql.DB) error {
	_, err := db.Exec(createTableQuery)
	return err
}
