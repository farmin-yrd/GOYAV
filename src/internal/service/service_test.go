package service

import (
	"bytes"
	"context"
	"goyav/internal/adapter/antivirus"
	"goyav/internal/adapter/storage/binaryrepo"
	"goyav/internal/adapter/storage/docrepo"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"goyav/pkg/helper"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	version           = "1.0"         // Version of the service
	info              = "information" // Basic information about the service
	resultTTL         = time.Second   // Time-to-live for results
	semaphoreCapacity = uint64(128)   // Capacity of the semaphore used in service
)

// TestServiceNew tests the New function of the service package
func TestServiceNew(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer
	)

	// Test case: Successful creation of a new service
	t.Run("Success", func(t *testing.T) {
		svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, resultTTL, semaphoreCapacity)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})

	// Test case: Attempt to create a new service with nil dependencies
	t.Run("NilDependencies", func(t *testing.T) {
		svc, err := New(nil, nil, nil, "1.0.0", "s Info", 24*time.Hour, 128)
		assert.Error(t, err)
		assert.Nil(t, svc)
	})

	// Test case: Simulate a ping failure in the dependencies
	t.Run("PingFailure", func(t *testing.T) {
		// Simulate ping failure
		binRepoMock.IsOnline(false)
		docRepoMock.IsOnline(false)
		antivirusMock.IsOnline(false)

		svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, resultTTL, semaphoreCapacity)

		// Verifying that the service initialization fails and then resetting mocks to their original state
		assert.Error(t, err)
		assert.Nil(t, svc)

		// switch on all instance
		binRepoMock.IsOnline(true)
		docRepoMock.IsOnline(true)
		antivirusMock.IsOnline(true)
	})

	// Test case for service initialization with insufficient semaphore capacity
	t.Run("InsufficientSemaphoreCapacity", func(t *testing.T) {
		svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, resultTTL, 0)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
		cap := uint64(cap(svc.semaphore))
		assert.Equal(t, DefaultSemaphoreCapacity, cap, "expected capacity=%d, got %d", DefaultSemaphoreCapacity, cap)
	})

	// Test case for service initialization with sufficient semaphore capacity
	t.Run("SufficientSemaphoreCapacity", func(t *testing.T) {
		s, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, resultTTL, DefaultSemaphoreCapacity+1)
		assert.NoError(t, err)
		assert.NotNil(t, s)
		cap := uint64(cap(s.semaphore))
		assert.Greater(t, cap, DefaultSemaphoreCapacity)
	})
}

func TestAutoPurge(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx = context.Background()
		ID  = "ITSzxj1mqz1gwFZ4iendeQ"
	)

	testDoc := &domain.Document{
		ID:         ID,
		Hash:       "hash",
		Tag:        "tag",
		Status:     domain.StatusClean,
		AnalyzedAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	err := docRepoMock.Save(ctx, testDoc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	waitDuration := resultTTL + time.Second
	t.Run("AutoPurgeDisabled", func(t *testing.T) {
		_, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, time.Duration(-1), semaphoreCapacity)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(waitDuration)
		doc, err := docRepoMock.Get(ctx, ID)
		assert.NoError(t, err, "no error is expected when getting doc")
		assert.NotNil(t, doc, "the saved document should be always present is the document repositorty")
	})

	t.Run("AutoPurgeEnabled", func(t *testing.T) {
		_, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, resultTTL, semaphoreCapacity)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(waitDuration)
		doc, err := docRepoMock.Get(ctx, ID)
		assert.ErrorIs(t, err, port.ErrDocumentNotFound, "the document should be deleted, and ErrDocumentNotFound is expected when getting doc")
		assert.Nil(t, doc, "the saved document should be deleted when auto purge enabled")
	})
}

// TestServiceVersion tests the version assignment in the service initialization.
func TestServiceVersion(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		expectedVersion = "1.0.0"
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, expectedVersion, "Service Info", 24*time.Hour, 128)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	version := svc.Version()
	assert.Equal(t, expectedVersion, version, "The actual version should match the expected version")
}

// TestServiceInformation tests the information assignment in the service initialization.
func TestServiceInformation(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		expectedInformation = "Service for managing documents"
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, "1.0.0", expectedInformation, 24*time.Hour, 128)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	information := svc.Information()
	assert.Equal(t, expectedInformation, information, "The actual information should match the expected information")
}

// TestServicePing tests the Ping function of the service to ensure it responds correctly.
func TestServicePing(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, resultTTL, semaphoreCapacity)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	t.Run("PingSuccess", func(t *testing.T) {
		err := svc.Ping()
		assert.NoError(t, err, "Ping should succeed when all dependencies are reachable")
	})

	// Test case: Simulate a failure in the service's Ping function
	t.Run("PingFailure", func(t *testing.T) {
		// Simulate ping failure
		binRepoMock.IsOnline(false)
		docRepoMock.IsOnline(false)
		antivirusMock.IsOnline(false)

		err := svc.Ping()
		assert.Error(t, err, "Ping should fail when one or more dependencies are unreachable")
	})
}

// TestServiceGetDocument tests the GetDocument function of the service for retrieving documents.
func TestServiceGetDocument(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ID           string
		testDocument = &domain.Document{
			Hash:       "hash",
			Tag:        "tag",
			Status:     domain.StatusClean,
			AnalyzedAt: time.Now(),
			CreatedAt:  time.Now(),
		}
	)
	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	// Test case for successfully retrieving a document
	t.Run("GetDocumentSuccess", func(t *testing.T) {
		ID = "ITSzxj1mqz1gwFZ4iendeQ"
		testDocument.ID = ID
		docRepoMock.Save(context.Background(), testDocument)
		document, err := svc.GetDocument(context.Background(), ID)
		assert.NoError(t, err)
		assert.Equal(t, testDocument, document)
	})

	t.Run("GetDocumentInvalidID", func(t *testing.T) {
		ID = "select * from document"
		document, err := svc.GetDocument(context.Background(), ID)
		assert.ErrorIs(t, err, port.ErrServiceInvalidID, "expected error ErrServiceGetDocumentFailed when the provided document ID is invalid")
		assert.Nil(t, document)
	})

	// Test case for handling the scenario where a document is not found
	t.Run("GetDocumentNotFound", func(t *testing.T) {
		ID = "xxxxXXXXxxxxXXXXxxxxXX"
		document, err := svc.GetDocument(context.Background(), ID)
		assert.ErrorIs(t, err, port.ErrServiceGetDocumentFailed, "expected error ErrServiceGetDocumentFailed when any document with the given ID does not exist in document repositoy")
		assert.Nil(t, document)
	})
}

// TestServiceUpload tests the Upload function of the service.
func TestUploadSuccessful(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx = context.Background()
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	mockData := bytes.NewReader(port.EICAR)
	size := int64(len(port.EICAR))

	// Test case for a successful file upload. It includes preparing mock data, expected tag, and status.

	providedTag := `EICAR - àéèêëïôù
		html: <div>tag</div>
		with json: {"documen": {"hello":"world"}}
		with sql: SELECT * FROM DOCUMENTS
		@%$<>*#`
	expectedStatus := domain.StatusInfected

	cw := helper.NewCryptoWriter()

	expectedHash, expectedID, err := cw.GenerateHashAndID(helper.Sanitize(providedTag))
	if err != nil {
		t.Fatalf("expected error: %v", err)
	}

	ID, err := svc.Upload(ctx, mockData, size, providedTag)

	assert.NoError(t, err, "no error should occur during a successful upload")
	assert.Equal(t, expectedID, ID, "expected ID=%q, got %q", expectedID, ID)

	doc, err := docRepoMock.Get(ctx, ID)
	assert.NoError(t, err, "retrieving the uploaded document should not produce an error")

	hash := doc.Hash
	assert.Equal(t, expectedHash, hash, "the hash of the retrieved document should match the expected hash")

	tag := doc.Tag
	l := len(tag)
	assert.Less(t, l, helper.TagMaxLength, "the length of the document tag should be less than or equal to the maximum allowed length")

	specialChars := []rune{':', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '-', '+', '"', '<', '>', '\'', '/', ' '}
	for c := range specialChars {
		assert.NotContains(t, tag, c, "the tag sould not contain special caracters")
	}

	assert.Contains(t, tag, "àéèêëïôù", "The tag may contain accented characters after sanitizing.")

	createAt := doc.CreatedAt
	assert.NotEmpty(t, createAt, "the creation time of the document should not be empty")

	// wait for antivirus anlysis to finish
	time.Sleep(time.Millisecond * 1500)

	status := doc.Status
	assert.Equal(t, expectedStatus, status, "the status of the retrieved document should match the expected status")

	analyzedAt := doc.AnalyzedAt
	assert.NotEmpty(t, analyzedAt, "the analysis datetime of the document should not be empty")

	reader, err := binRepoMock.Get(ctx, expectedID)
	assert.Error(t, err, "an error is expected when retrieving a document by its ID after analysis is completed")
	assert.Nil(t, reader, "a nil reader is expected when retrieving a document by its ID after analysis is completed")
}

// Test case for re-uploading an existing document with an empty tag
// in this cas no new document will be created: we return the ID of the existing document
func TestUploadDocumentWithEmptyTag(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx      = context.Background()
		mockData = bytes.NewReader(port.EICAR)
		size     = int64(len(port.EICAR))
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = svc.DocumentRepository.Purge(time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ID, err := svc.Upload(ctx, mockData, size, "")
	assert.NoError(t, err, "no error expected for a successful upload")
	assert.NotEmpty(t, ID, "an ID is expected after a successful upload")
}

// Test case for re-uploading an existing document the same tags
func TestReUploadExistingDocumentWithSameTag(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx      = context.Background()
		mockData = bytes.NewReader(port.EICAR)
		size     = int64(len(port.EICAR))
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tag := "EICAR"
	ID, err := svc.Upload(ctx, mockData, size, tag)
	assert.NoError(t, err, "no error expected for a successful upload")
	assert.NotEmpty(t, ID, "an ID is expected after a successful upload")

	// reupload the same document with the same tag
	newID, err := svc.Upload(ctx, mockData, size, tag)
	assert.ErrorIs(t, err, port.ErrDocumentAlreadyExists, "ErrDocumentAlreadyExists expected for uploading an existing document")
	assert.Equal(t, ID, newID, "the ID for the re-uploaded document should match the original upload ID")
}

// Test case for re-uploading an existing document with different tags, when the existing document's status is pending
// in this case a new document should be created with a new ID and a different creation datetime. The Hash and Analysis status of both documents should
// sould be identical.
func TestReUploadExistingDocumentStatusPendingWithNewTag(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx      = context.Background()
		mockData = port.EICAR
		size     = int64(len(port.EICAR))
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ID, err := svc.Upload(ctx, bytes.NewReader(mockData), size, "EICAR")
	assert.NoError(t, err, "no error expected for a successful upload")
	assert.NotEmpty(t, ID, "an ID is expected after a successful upload")

	// Re-upload the same document with a different tag, expecting a different ID
	newID, err := svc.Upload(ctx, bytes.NewReader(mockData), size, "eicar")
	assert.NoError(t, err, "no error expected for uploading an existing document")
	assert.NotEqual(t, ID, newID, "a different ID is expected when the tag is changed")

	existingDoc, err := svc.DocumentRepository.Get(ctx, ID)
	if err != nil {
		t.Fatalf("unexpected error occurred when retrieving the document: %v", err)
	}

	assert.Equal(t, existingDoc.Status, domain.StatusPending, "the status of the original document should be 'pending'")
	newDoc, err := svc.DocumentRepository.Get(ctx, newID)
	if err != nil {
		t.Fatalf("unexpected error occurred when retrieving the document: %v", err)
	}

	assert.NotEqual(t, newDoc.ID, existingDoc.ID, "the ID of the re-uploaded document should be different from the existing document")
	assert.NotEqual(t, existingDoc.CreatedAt, newDoc.CreatedAt, "expected diffrent creation dates")

	// wait for antivirus anlysis to finish
	time.Sleep(time.Millisecond * 1500)

	assert.NotEqual(t, existingDoc.Status, domain.StatusPending, "the status of the existing document should not be 'pending' after analysis")
	assert.NotEqual(t, newDoc.Status, domain.StatusPending, "the status of the re-uploaded document should not be 'pending' after analysis")

	assert.Equal(t, existingDoc.Hash, newDoc.Hash, "expected the same analysis hash")
	assert.Equal(t, existingDoc.Status, newDoc.Status, "the analysis status of both documents should be the same")
}

// Test case for re-uploading an existing document with different tags when the existing document status is not pending
// in this case a new document with a new ID should be created. The hash, analysis status, and analysis data time sould be the same for both documents.
func TestReUploadExistingDocumentStatusNotPendingWithNewTag(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx      = context.Background()
		mockData = port.EICAR
		size     = int64(len(port.EICAR))
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ID, err := svc.Upload(ctx, bytes.NewReader(mockData), size, "EICAR")
	assert.NoError(t, err, "no error is expected for the initial successful upload")
	assert.NotEmpty(t, ID, "an ID is expected after the initial successful upload")

	// wait for antivirus anlysis to finish
	time.Sleep(time.Millisecond * 1500)

	// reupload the same document with a different tag
	newID, err := svc.Upload(ctx, bytes.NewReader(mockData), size, "eicar")
	assert.ErrorIs(t, err, port.ErrDocumentAlreadyExists, "ErrDocumentAlreadyExists expected for uploading an existing document")
	assert.NotEqual(t, ID, newID, "expected diffrent ID, got the same: %q", ID)

	doc, err := svc.DocumentRepository.Get(ctx, ID)
	if err != nil {
		t.Fatalf("unexpected error occurred when retrieving the document: %v", err)
	}

	assert.NotEqual(t, doc.Status, domain.StatusPending, "the status of the original document should not be 'pending'")
	newDoc, err := svc.DocumentRepository.Get(ctx, newID)
	if err != nil {
		t.Fatalf("unexpected error occurred when retrieving the document: %v", err)
	}

	assert.NotEqual(t, newDoc.Status, domain.StatusPending, "the status of the re-uploaded document should be 'pending'")

	assert.NotEqual(t, doc.CreatedAt, newDoc.CreatedAt, "the creation dates of the original and re-uploaded documents should be different")
	assert.Equal(t, doc.Hash, newDoc.Hash, "expected the same analysis hash")
	assert.Equal(t, doc.AnalyzedAt, newDoc.AnalyzedAt, "expected the same analysis date")
	assert.Equal(t, doc.Status, newDoc.Status, "expected the same analysis status")
}

// TestUploadUnavailableDependencies checks the Upload function's behavior when dependencies
// like document and binary repositories, and the antivirus service are unavailable.
func TestUploadUnvailableDependencies(t *testing.T) {
	var (
		binRepoMock   = binaryrepo.NewMock() // binary repository
		docRepoMock   = docrepo.NewMock()    // document repository
		antivirusMock = antivirus.NewMock()  // antivirus analyzer

		ctx      = context.Background()
		mockData = bytes.NewReader(port.EICAR)
		size     = int64(len(port.EICAR))
	)

	svc, err := New(binRepoMock, docRepoMock, antivirusMock, version, info, 0, semaphoreCapacity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Case: Upload Failure Due to Document Repository Unavailability
	t.Run("DocumentRepositoryUnavailable", func(t *testing.T) {
		// put document repository offline
		docRepoMock.IsOnline(false)
		ID, err := svc.Upload(ctx, mockData, size, "EICAR")
		assert.Error(t, err, "error expected when document repository is unavailable")
		assert.Empty(t, ID, "empty ID expected when document repository is unavailable")
		// put document repository online
		docRepoMock.IsOnline(true)
	})

	// Test Case: Upload Failure Due to Binary Repository Unavailability
	t.Run("BinaryRepositoryUnavailable", func(t *testing.T) {
		// put binary repository offline
		binRepoMock.IsOnline(false)
		ID, err := svc.Upload(ctx, mockData, size, "EICAR")
		assert.Error(t, err, "error expected when document repository is unavailable")
		assert.Empty(t, ID, "empty ID expected when document repository is unavailable")
		// put binary repository online
		binRepoMock.IsOnline(true)
	})

	// Test Case: Handling Antivirus Service Temporary Unavailability During Upload
	t.Run("AntivirusServiceTemporaryUnavailable", func(t *testing.T) {
		// put analyser offline
		antivirusMock.IsOnline(false)
		ID, err := svc.Upload(ctx, mockData, mockData.Size(), "EICAR")
		assert.NoError(t, err, "error expected when antivirus analyser is unavailable")
		assert.NotEmpty(t, ID, "empty ID expected when antivirus is unavailable")

		doc, err := svc.DocumentRepository.Get(ctx, ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assert.Equal(t, domain.StatusPending, doc.Status, "expected status pending when antivirus is offline just after upload")

		// put the analyzer online
		antivirusMock.IsOnline(true)

		// wait at least 5 seconds for a new analysis attemp
		time.Sleep(8 * time.Second)
		assert.NotEqual(t, domain.StatusPending, doc.Status, "expected status update after a new analysis attemp")

		analyzedAt := doc.AnalyzedAt
		assert.NotEmpty(t, analyzedAt, "expected analyzedAt updated after a new analyze attemp")
	})
}
