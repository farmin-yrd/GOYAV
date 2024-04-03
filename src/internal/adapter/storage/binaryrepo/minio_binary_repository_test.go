package binaryrepo

import (
	"bytes"
	"context"
	"fmt"
	"goyav/pkg/helper"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	container testcontainers.Container
	client    *minio.Client
	ctx       = context.Background()
)

func TestMain(m *testing.M) {
	var (
		minioPort uint64 = 9000
		host      string
		err       error
	)
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{fmt.Sprintf(`%v/tcp`, minioPort)},
		Env:          map[string]string{"MINIO_ACCESS_KEY": "minioadmin", "MINIO_SECRET_KEY": "minioadmin"},
		Cmd:          []string{"server", "/data"},
		WaitingFor:   wait.ForHTTP("/minio/health/live").WithPort("9000/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, host, err = helper.SetupContainer(ctx, req)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	minioEndpoint := fmt.Sprintf("%v:%v", host, minioPort)
	client, err = minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	c := m.Run()

	if err := container.Terminate(ctx); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	os.Exit(c)
}

func TestNewMinio(t *testing.T) {
	tests := []struct {
		name       string
		client     *minio.Client
		bucketName string
		wantErr    bool
	}{
		{
			name:       "Valid client and bucket",
			client:     client,
			bucketName: "test-bucket",
			wantErr:    false,
		},
		{
			name:       "Nil client",
			client:     nil,
			bucketName: "test-bucket",
			wantErr:    true,
		},
		{
			name:       "Empty bucket name",
			client:     client,
			bucketName: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMinio(tt.client, tt.bucketName)
			if tt.wantErr {
				assert.Error(t, err, "Expected an error for %s", tt.name)
			} else {
				assert.NoError(t, err, "Expected no error for %s", tt.name)
			}
		})
	}
}

func TestSave(t *testing.T) {
	bucketName := "test-bucket"
	// Create a new instance of MinioBinaryRepository
	repo, err := NewMinio(client, bucketName)
	if err != nil {
		t.Fatalf("Failed to create MinioBinaryRepository: %v", err)
	}

	// Define test data
	testID := "test-file"
	testData := []byte("Hello, MinIO!")
	ctx := context.Background()

	// Test the Save function
	err = repo.Save(ctx, bytes.NewReader(testData), int64(len(testData)), testID)
	if err != nil {
		t.Errorf("Failed to save data: %v", err)
	}

	// Verify if the data was saved correctly
	object, err := client.GetObject(ctx, bucketName, testID, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("Failed to retrieve saved data: %v", err)
	}
	defer object.Close()

	retrievedData, err := io.ReadAll(object)
	if err != nil {
		t.Fatalf("Failed to read saved data: %v", err)
	}

	// Assert if the saved data is equal to the test data
	assert.Equal(t, testData, retrievedData, "Retrieved data should match the test data")
}

func TestDelete(t *testing.T) {
	bucketName := "test-bucket"
	// Create a new instance of MinioBinaryRepository
	repo, err := NewMinio(client, bucketName)
	if err != nil {
		t.Fatalf("Failed to create MinioBinaryRepository: %v", err)
	}

	ctx := context.Background()

	// Sub-test 1: Delete existing data
	t.Run("DeleteExistingData", func(t *testing.T) {
		// Define and save test data
		testID := "test-file"
		testData := []byte("Hello, MinIO!")
		err = repo.Save(ctx, bytes.NewReader(testData), int64(len(testData)), testID)
		if err != nil {
			t.Fatalf("Failed to save data: %v", err)
		}

		err := repo.Delete(ctx, testID)
		assert.NoError(t, err, "Delete method should not return an error for existing data")

		// Verify if the data was deleted correctly
		// Directly check with MinIO client if the file has been deleted
		_, err = client.StatObject(ctx, bucketName, testID, minio.StatObjectOptions{})
		assert.Error(t, err, "StatObject should return an error for deleted data")
	})

	// Sub-test 2: Delete non-existing data
	t.Run("DeleteNonExistingData", func(t *testing.T) {
		nonExistingID := "non-existing-file"
		err := repo.Delete(ctx, nonExistingID)
		assert.Error(t, err, "Delete method should not return an error for non-existing data")
	})
}

func TestGet(t *testing.T) {

	// Create a new instance of MinioBinaryRepository
	bucketName := "test-bucket"
	repo, err := NewMinio(client, bucketName)
	if err != nil {
		t.Fatalf("Failed to create MinioBinaryRepository: %v", err)
	}

	ctx := context.Background()

	// Test case 1: Get an existing file
	t.Run("GetExistingFile", func(t *testing.T) {
		testID := "test-file"
		testData := []byte("Hello, MinIO!")

		// Save data first
		err := repo.Save(ctx, bytes.NewReader(testData), int64(len(testData)), testID)
		if err != nil {
			t.Fatalf("Failed to save data: %v", err)
		}

		// Get the saved data
		readCloser, err := repo.Get(ctx, testID)
		assert.NoError(t, err, "Get should not return an error for existing data")
		defer readCloser.Close()

		retrievedData, err := io.ReadAll(readCloser)
		assert.NoError(t, err, "ReadAll should not return an error for existing data")
		assert.Equal(t, testData, retrievedData, "Retrieved data should match saved data")
	})

	// Test case 2: Get a non-existing file
	t.Run("GetNonExistingFile", func(t *testing.T) {
		nonExistingID := "non-existing-file"

		// Attempt to get non-existing data
		_, err := repo.Get(ctx, nonExistingID)
		assert.Error(t, err, "Get should return an error for non-existing data")
	})
}

func TestPing(t *testing.T) {
	bucketName := "test-bucket"

	repo, err := NewMinio(client, bucketName)
	if err != nil {
		t.Fatalf("Failed to create MinioBinaryRepository: %v", err)
	}

	// Test Ping for a working MinIO instance
	t.Run("PingSuccess", func(t *testing.T) {
		err = repo.Ping()
		assert.NoError(t, err, "Ping should not return an error for a working MinIO instance")
	})

	// Test Ping for a stopped MinIO instance
	t.Run("PingFailure", func(t *testing.T) {
		stopDuration := time.Second * 2
		container.Stop(ctx, &stopDuration)
		err = repo.Ping()
		assert.Error(t, err, "Ping should return an error for a stopped MinIO instance")
		container.Start(ctx)
	})
}
