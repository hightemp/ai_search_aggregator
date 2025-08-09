//go:build integration
// +build integration

package main

import (
	"context"
	"net/http"
	"testing"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestSearxContainer(t *testing.T) {
	ctx := context.Background()

	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "searxng/searxng:latest",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/search?q=test&format=json").WithStartupTimeout(120e9),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("container start: %v", err)
	}
	defer c.Terminate(ctx)

	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "8080")

	url := "http://" + host + ":" + port.Port() + "/search?q=golang&format=json"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}
