package antivirus

import (
	"bytes"
	"context"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"goyav/pkg/helper"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	container testcontainers.Container
	req       testcontainers.ContainerRequest
	ctx       = context.Background()

	timeout    uint64 = 10
	clamavPort uint64 = 3310
	clamavHost string
)

func TestMain(m *testing.M) {

	var err error
	req = testcontainers.ContainerRequest{
		Image:        "clamav/clamav:1.3",
		ExposedPorts: []string{fmt.Sprintf("%v", clamavPort)},
		WaitingFor:   wait.ForLog("socket found, clamd started."),
	}

	container, clamavHost, err = helper.SetupContainer(ctx, req)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	c := m.Run()

	if err := container.Terminate(ctx); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	os.Exit(c)
}

func TestNewClamav(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		_, err := NewClamav(clamavHost, clamavPort, timeout)
		assert.NoError(t, err, "Expected no error but got %v", err)
	})

	t.Run("InvalideTimeout", func(t *testing.T) {
		_, err := NewClamav(clamavHost, clamavPort, 0)
		assert.Error(t, err)
	})

}

func TestClamavAnalyser_Analyze(t *testing.T) {
	analyser, err := NewClamav(clamavHost, clamavPort, timeout)
	assert.NoError(t, err)

	t.Run("CleanData", func(t *testing.T) {
		data := bytes.NewReader([]byte("clean data"))
		ctx := context.Background()
		status, err := analyser.Analyze(ctx, data)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusClean, status)
	})

	t.Run("InfectedData", func(t *testing.T) {
		data := bytes.NewReader(port.EICAR)
		ctx := context.Background()
		status, err := analyser.Analyze(ctx, data)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusInfected, status)
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		data := bytes.NewReader([]byte("clean data"))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := analyser.Analyze(ctx, data)
		assert.Error(t, err)
	})
}

func TestClamavAnalyser_Ping(t *testing.T) {
	analyser, err := NewClamav(clamavHost, clamavPort, timeout)
	assert.NoError(t, err)

	t.Run("PingSuccess", func(t *testing.T) {
		err := analyser.Ping()
		assert.NoError(t, err)
	})

	t.Run("PingFailure", func(t *testing.T) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		stopDuration := time.Second * 2
		container.Stop(ctx, &stopDuration)
		err = analyser.Ping()
		assert.Error(t, err)
		container.Start(ctx)
	})
}
