package main

import (
	"database/sql"
	"errors"
	"fmt"
	"goyav/internal/adapter/antivirus"
	"goyav/internal/adapter/storage/binaryrepo"
	"goyav/internal/adapter/storage/docrepo"
	"goyav/internal/core/port"
	"goyav/internal/service"
	"goyav/pkg/helper"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	// Default upload size limit in bytes : 1 Mib
	DefaultMaxUploadSize    uint64        = 1 << 20
	DefaultUploadTimeout    uint64        = 10
	DefaultResultTimeToLive time.Duration = time.Hour
)

// setup initializes the GoyAV application with necessary configurations.
// It configures the host, port, max upload size, version, and information for the application,
// along with initializing byte repository, document repository and antivirus analyzer
func setup(host *string, port *int64, maxUploadSize *uint64, uploadTimeout *uint64, ver *string, info *string, resTTL *time.Duration, semaphoreCapacity *uint64, b *port.BinaryRepository, d *port.DocumentRepository, a *port.AntivirusAnalyzer) error {
	var err error

	setLogger()

	// Configure host (default: localhost) and port (default : 80)
	*host = helper.GetEnvWithDefault("GOYAV_HOST", "localhost")
	*port, err = strconv.ParseInt(helper.GetEnvWithDefault("GOYAV_PORT", "80"), 10, 64)
	if err != nil {
		return errors.New("GOYAV_PORT must be a valid port number")
	}
	slog.Info("server configuration", "host", *host, "port", *port)

	// Configure version
	*ver, err = helper.GetEnvWithError("GOYAV_VERSION")
	if err != nil {
		return errors.New("GOYAV_VERSION must be set")
	}
	slog.Info("application version set", "version", *ver)

	// Configure information (default: GoyAV)
	*info = helper.GetEnvWithDefault("GOYAV_INFORMATION", "GoyAV")
	slog.Info("application information set", "information", *info)

	// Configure maximum upload size (default: 1 MiB)
	*maxUploadSize, err = strconv.ParseUint(helper.GetEnvWithDefault("GOYAV_MAX_UPLOAD_SIZE", ""), 10, 64)
	if err != nil || *maxUploadSize == 0 {
		*maxUploadSize = DefaultMaxUploadSize
		slog.Warn("setting maximum upload size set to default", "default (bytes)", *maxUploadSize)
	}
	slog.Info("maximum upload size set", "size (bytes)", *maxUploadSize)

	// Configure upload timeout in seconds (default: 10 seconds)
	*uploadTimeout, err = strconv.ParseUint(helper.GetEnvWithDefault("GOYAV_UPLOAD_TIMEOUT", ""), 10, 64)
	if err != nil || *uploadTimeout <= 0 {
		*uploadTimeout = DefaultUploadTimeout
		slog.Warn("setting upload timeout to default", "default (seconds)", DefaultUploadTimeout)
	}
	slog.Info("upload timeout set", "timeout (seconds)", uploadTimeout)

	// Configure result time to live (default: 1 hour)
	*resTTL, err = time.ParseDuration(helper.GetEnvWithDefault("GOYAV_RESULT_TTL", "1h"))
	if err != nil {
		*resTTL = DefaultResultTimeToLive
		slog.Warn("setting result time to live to default", "default", resTTL.String())
	}
	slog.Info("result time to live set", "duration", (*resTTL).String())
	slog.Info("document repository auto-purge set", "auto-purge ?", *resTTL > 0)

	// Configure semaphore capacity (default: 128 goroutines)
	*semaphoreCapacity, err = strconv.ParseUint(helper.GetEnvWithDefault("GOYAV_SEMAPHORE_CAPACITY", "128"), 10, 64)
	if err != nil {
		*semaphoreCapacity = service.DefaultSemaphoreCapacity
		slog.Warn("setting semaphore capacity to default", "default", "128 goroutines")
	}
	slog.Info("semaphore capacity set", "capacity (goroutines)", semaphoreCapacity)

	// Initialize byte repository
	if err = setupMinioByteRepository(b); err != nil {
		return fmt.Errorf("error while creating binary repository: %w", err)
	}

	// Initialize document repository
	if err = setupPostgresDocumentRepository(d); err != nil {
		return fmt.Errorf("error while creating document repository: %w", err)
	}

	// Initialize antivirus analyzer
	if err = setupClamAVAnalyzer(a); err != nil {
		return fmt.Errorf("error while creating antivirus analyzer: %w", err)
	}

	return nil
}

// setupMinioByteRepository configures a s3 binary repository for storing binary data of files.
func setupMinioByteRepository(b *port.BinaryRepository) error {
	var err error

	// Retrieve the s3 endpoint endpoint : host and port without protocol
	endpoint, err := helper.GetEnvWithError("GOYAV_S3_ENDPOINT_URL")
	if err != nil {
		return err
	}
	slog.Info("configuring s3 bucket", "endpoint URL", endpoint)

	// Retrieve s3 access key ID with error check
	accessKeyID, err := helper.GetEnvWithError("GOYAV_S3_ACCESS_KEY")
	if err != nil {
		return err
	}
	slog.Info("configuring s3 bucket", "access key ID", accessKeyID)

	// Retrieve s3 secret key with error check
	secretKey, err := helper.GetEnvWithError("GOYAV_S3_SECRET_KEY")
	if err != nil {
		return err
	}
	slog.Debug("configuring s3 bucket", "secret key", secretKey)

	// Retrieve s3 bucket name configuration
	bucketName := helper.GetEnvWithDefault("GOYAV_S3_BUCKET_NAME", "goyav")
	slog.Info("configuring s3 bucket", "bucket name", bucketName)

	// Parse and validate s3 SSL usage
	useSSL, err := strconv.ParseBool(helper.GetEnvWithDefault("GOYAV_S3_USE_SSL", "false"))
	if err != nil {
		return errors.New("GOYAV_S3_USE_SSL must be true or false")
	}
	slog.Info("configuring s3 bucket", "use ssl ?", useSSL)

	// Create s3 client
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return err
	}

	*b, err = binaryrepo.NewMinio(cli, bucketName)
	if err != nil {
		return err
	}

	slog.Info("minio repository setup complete")
	return nil
}

// setupPostgresDocumentRepository configures a Postgres document repository.
func setupPostgresDocumentRepository(d *port.DocumentRepository) error {
	var err error

	// Retrieve PostgreSQL hst configuration
	hst := helper.GetEnvWithDefault("GOYAV_POSTGRES_HOST", "127.0.0.1")
	slog.Info("configuring postgres", "host", hst)

	// Parse and validate PostgreSQL port
	prt, err := strconv.ParseUint(helper.GetEnvWithDefault("GOYAV_POSTGRES_PORT", "5432"), 10, 64)
	if err != nil {
		return errors.New("GOYAV_POSTGRES_PORT must be a valid port number")
	}
	slog.Info("configuring postgres", "port", prt)

	// Retrieve PostgreSQL user
	user, err := helper.GetEnvWithError("GOYAV_POSTGRES_USER")
	if err != nil {
		return fmt.Errorf("GOYAV_POSTGRES_USER must be a valid user name: %w", err)
	}
	slog.Info("configuring postgres", "user", user)

	// Retrieve PostgreSQL user passwd
	passwd, err := helper.GetEnvWithError("GOYAV_POSTGRES_USER_PASSWORD")
	if err != nil {
		return fmt.Errorf("GOYAV_POSTGRES_USER_PASSWORD is not valid: %w", err)
	}
	slog.Debug("configuring postgres", "password", passwd)

	// Retrieve PostgreSQL database name
	dbname, err := helper.GetEnvWithError("GOYAV_POSTGRES_DB")
	if err != nil {
		return fmt.Errorf("GOYAV_POSTGRES_DB is not valid : %w", err)
	}
	slog.Info("configuring postgres", "database name", dbname)

	// Retrieve PostgreSQL schema
	schema, err := helper.GetEnvWithError("GOYAV_POSTGRES_SCHEMA")
	if err != nil {
		return fmt.Errorf("GOYAV_POSTGRES_SCHEMA is not valid: %w", err)
	}
	slog.Info("configuring postgres", "postgres schema name", schema)

	// Retrieve PostgreSQL SSL usage
	ssl := helper.GetEnvWithDefault("GOYAV_POSTGRES_SSL_MODE", "require")
	slog.Info("configuring postgres", "postgres ssl mode", ssl)

	connInfo := fmt.Sprintf("host=%v port=%v dbname=%v search_path=%v sslmode=%v user=%v password=%v", hst, prt, dbname, schema, ssl, user, passwd)
	db, err := sql.Open("postgres", connInfo)

	if err != nil {
		return err
	}

	// Initialize the PostgreSQL document repository
	*d, err = docrepo.NewPotgres(db)
	if err != nil {
		return err
	}

	slog.Info("postgres repository setup complete")
	return nil
}

// setupClamAVAnalyzer configures a ClamAV antivirus analyzer.
func setupClamAVAnalyzer(a *port.AntivirusAnalyzer) error {
	var err error

	// Retrieve ClamAV host configuration
	clamdHost := helper.GetEnvWithDefault("GOYAV_CLAMAV_HOST", "127.0.0.1")
	slog.Info("configuring clamav", "host", clamdHost)

	// Parse and validate ClamAV port
	clamdPort, err := strconv.ParseUint(helper.GetEnvWithDefault("GOYAV_CLAMAV_PORT", "3310"), 10, 64)
	if err != nil {
		return errors.New("GOYAV_CLAMAV_PORT must be a valid port number")
	}
	slog.Info("configuring clamav", "port", clamdPort)

	// Parse and validate ClamAV timeout
	clamdTimeout, err := strconv.ParseUint(helper.GetEnvWithDefault("GOYAV_CLAMAV_TIMEOUT", "30"), 10, 64)
	if err != nil {
		return errors.New("GOYAV_CLAMAV_TIMEOUT must be a strictly positive number")
	}
	slog.Info("configuring clamav", "timeout", clamdTimeout)

	// Initialize the ClamAV analyzer
	*a, err = antivirus.NewClamav(clamdHost, clamdPort, clamdTimeout)
	if err != nil {
		return err
	}

	slog.Info("clamav analyzer setup complete")
	return nil
}

func setLogger() {
	var level slog.Level = slog.LevelInfo

	isDubugMode, _ := strconv.ParseBool(helper.GetEnvWithDefault("GOYAV_DEBUG_MODE", "false"))
	if isDubugMode {
		level = slog.LevelDebug
	}

	slog.SetDefault(
		slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: level,
			}),
		),
	)
}
