package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/LumiWave/lumiwave-protocol/app"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultGRPCEndpoint = "localhost:9090"
	defaultTimeout      = 7 * time.Second
)

// newGRPCConn creates a gRPC client connection.
func newGRPCConn(endpoint string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect gRPC endpoint (%s): %w", endpoint, err)
	}
	return conn, nil
}

// newTimeoutContext creates a context with timeout.
func newTimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// printJSON prints a value as pretty JSON.
func printJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

// exitErr prints an error and exits the process.
func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
