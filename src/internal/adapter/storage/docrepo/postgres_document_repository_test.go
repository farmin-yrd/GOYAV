package docrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNewPotgresDocumentRepository(t *testing.T) {
	t.Run("SuccessfulCreation", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("error creating sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(1, 1))

		repo, err := NewPotgres(db)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("NilDatabase", func(t *testing.T) {
		repo, err := NewPotgres(nil)
		expectedError := fmt.Errorf("%w : required sql.DB, got nil", ErrPostgresDocumentRepository)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Nil(t, repo)
	})

	t.Run("DatabaseConnectionFailure", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("error creating sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(fmt.Errorf("database connection error"))

		repo, err := NewPotgres(db)
		assert.Error(t, err)
		assert.Nil(t, repo)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("DocumentTableSetupFailure", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("error creating sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS").WillReturnError(fmt.Errorf("error setting up document table"))

		repo, err := NewPotgres(db)
		assert.Error(t, err)
		assert.Nil(t, repo)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("FAIL : an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}

	doc := &domain.Document{
		ID:         "123",
		Hash:       "abc123",
		Tag:        "example",
		Status:     1,
		AnalyzedAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	t.Run("SuccessfulSave", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO documents").
			WithArgs(doc.ID, doc.Hash, doc.Tag, doc.Status, doc.AnalyzedAt, doc.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Save(context.Background(), doc)
		assert.NoError(t, err)
	})

	t.Run("SaveWithAlreadyExistingDocument", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO documents").
			WithArgs(doc.ID, doc.Hash, doc.Tag, doc.Status, doc.AnalyzedAt, doc.CreatedAt).
			WillReturnError(sql.ErrNoRows) // Simulating a unique constraint violation

		err := repo.Save(context.Background(), doc)
		assert.Error(t, err)
	})

	t.Run("DatabaseErrorOnSave", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO documents").
			WithArgs(doc.ID, doc.Hash, doc.Tag, doc.Status, doc.AnalyzedAt, doc.CreatedAt).
			WillReturnError(sql.ErrConnDone) // Simulating a database connection error

		err := repo.Save(context.Background(), doc)
		assert.Error(t, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}

	t.Run("DocumentFound", func(t *testing.T) {
		docID := "123"
		rows := sqlmock.NewRows([]string{"document_id", "hash", "tag", "status", "analyzed_at", "created_at"}).
			AddRow(docID, "hash123", "tag1", 1, time.Now(), time.Now())

		mock.ExpectQuery("SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE document_id =").
			WithArgs(docID).
			WillReturnRows(rows)

		doc, err := repo.Get(context.Background(), docID)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
	})

	t.Run("DocumentNotFound", func(t *testing.T) {
		docID := "unknown"
		mock.ExpectQuery("SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE document_id =").
			WithArgs(docID).
			WillReturnError(sql.ErrNoRows)

		doc, err := repo.Get(context.Background(), docID)
		assert.Error(t, err)
		assert.Nil(t, doc)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		docID := "error"
		mock.ExpectQuery("SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE document_id =").
			WithArgs(docID).
			WillReturnError(sql.ErrConnDone)

		doc, err := repo.Get(context.Background(), docID)
		assert.Error(t, err)
		assert.Nil(t, doc)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetByHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}

	t.Run("DocumentFound", func(t *testing.T) {
		docHash := "hash123"
		rows := sqlmock.NewRows([]string{"document_id", "hash", "tag", "status", "analyzed_at", "created_at"}).
			AddRow("123", docHash, "tag1", 1, time.Now(), time.Now())

		mock.ExpectQuery("SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE hash =").
			WithArgs(docHash).
			WillReturnRows(rows)

		doc, err := repo.GetByHash(context.Background(), docHash)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, docHash, doc.Hash)
	})

	t.Run("DocumentNotFound", func(t *testing.T) {
		docHash := "unknownhash"
		mock.ExpectQuery("SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE hash =").
			WithArgs(docHash).
			WillReturnError(sql.ErrNoRows)

		doc, err := repo.GetByHash(context.Background(), docHash)
		assert.Error(t, err)
		assert.Nil(t, doc)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		docHash := "errorhash"
		mock.ExpectQuery("SELECT document_id, hash, tag, status, analyzed_at, created_at FROM documents WHERE hash =").
			WithArgs(docHash).
			WillReturnError(sql.ErrConnDone)

		doc, err := repo.GetByHash(context.Background(), docHash)
		assert.Error(t, err)
		assert.Nil(t, doc)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}

	// Test case: Successful deletion
	t.Run("SuccessfulDeletion", func(t *testing.T) {
		docID := "123"
		mock.ExpectExec("DELETE FROM documents WHERE document_id =").
			WithArgs(docID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Delete(context.Background(), docID)
		assert.NoError(t, err)
	})

	// Test case: Deleting a non-existent document
	t.Run("NonExistentDocument", func(t *testing.T) {
		docID := "nonexistent"
		mock.ExpectExec("DELETE FROM documents WHERE document_id =").
			WithArgs(docID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected

		err := repo.Delete(context.Background(), docID)
		// Assuming an error is expected when trying to delete a non-existent document
		assert.Error(t, err)
	})

	// Test case: Database error during deletion
	t.Run("DatabaseError", func(t *testing.T) {
		docID := "dbError"
		mock.ExpectExec("DELETE FROM documents WHERE document_id =").
			WithArgs(docID).
			WillReturnError(errors.New("database error"))

		err := repo.Delete(context.Background(), docID)
		assert.Error(t, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}

	// Scenario: Successfully updating a document's status
	t.Run("StatusUpdated", func(t *testing.T) {
		docID := "123"
		newStatus := domain.StatusClean
		analyzedAt := time.Now()

		mock.ExpectExec("UPDATE documents SET status = .+, analyzed_at = .+ WHERE document_id = .+").
			WithArgs(newStatus, sqlmock.AnyArg(), docID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateStatus(context.Background(), docID, newStatus, analyzedAt)
		assert.NoError(t, err)
	})

	// Scenario: Trying to update a non-existing document
	t.Run("DocumentNotFound", func(t *testing.T) {
		docID := "nonexistent"
		newStatus := domain.StatusClean
		analyzedAt := time.Now()

		mock.ExpectExec("UPDATE documents SET status = .+, analyzed_at = .+ WHERE document_id = .+").
			WithArgs(newStatus, sqlmock.AnyArg(), docID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected

		err := repo.UpdateStatus(context.Background(), docID, newStatus, analyzedAt)
		assert.Error(t, err)
	})

	// Scenario: Encountering a database error during update
	t.Run("DatabaseError", func(t *testing.T) {
		docID := "errorcase"
		newStatus := domain.AnalysisStatus(2)
		analyzedAt := time.Now()

		mock.ExpectExec("UPDATE documents SET status = .+, analyzed_at = .+ WHERE document_id = .+").
			WithArgs(newStatus, sqlmock.AnyArg(), docID).
			WillReturnError(sql.ErrConnDone) // Simulating a database error

		err := repo.UpdateStatus(context.Background(), docID, newStatus, analyzedAt)
		assert.Error(t, err)
	})

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestPurge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}
	purgeTime := time.Now().Add(-24 * time.Hour) // Purging documents older than 24 hours

	// Scenario: Successfully purging the documents
	t.Run("SuccessfulPurge", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM documents WHERE created_at < \\$1 AND status != \\$2").
			WithArgs(purgeTime, domain.StatusPending).
			WillReturnResult(sqlmock.NewResult(0, 1)) // Simulating one row affected

		err := repo.Purge(purgeTime)
		assert.NoError(t, err)
	})

	// Scenario: Encountering a database error during purge
	t.Run("DatabaseError", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM documents WHERE created_at < \\$1 AND status != \\$2").
			WithArgs(purgeTime, domain.StatusPending).
			WillReturnError(sql.ErrConnDone) // Simulating a database error

		err := repo.Purge(purgeTime)
		assert.Error(t, err)
	})

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestPing(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresDocumentRepository{db: db}

	// Scenario: Successful Ping to the database
	t.Run("SuccessfulPing", func(t *testing.T) {
		mock.ExpectPing().WillReturnError(nil)

		err := repo.Ping()
		assert.NoError(t, err)
	})

	// Scenario: Ping fails due to database connection error
	t.Run("FailedPing", func(t *testing.T) {
		mock.ExpectPing().WillReturnError(sql.ErrConnDone)

		err := repo.Ping()
		assert.Error(t, err)
	})

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
