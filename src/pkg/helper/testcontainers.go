package helper

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
)

func SetupContainer(ctx context.Context, req testcontainers.ContainerRequest) (testcontainers.Container, string, error) {

	cnt, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	ip, err := cnt.ContainerIP(ctx)
	if err != nil {
		return nil, "", err
	}

	return cnt, ip, nil
}
