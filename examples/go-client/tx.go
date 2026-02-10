package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"strings"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// runTxCommand routes tx subcommands.
func runTxCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("tx subcommand is required (send/send-privkey/get/search/simulate/broadcast)")
	}

	switch args[0] {
	case "send":
		return runTxSend(args[1:])
	case "send-privkey":
		return runTxSendWithPrivateKey(args[1:])
	case "get":
		return runTxGet(args[1:])
	case "search":
		return runTxSearch(args[1:])
	case "simulate":
		return runTxSimulate(args[1:])
	case "broadcast":
		return runTxBroadcast(args[1:])
	default:
		return fmt.Errorf("unsupported tx subcommand: %s", args[0])
	}
}

// runTxGet fetches a transaction by hash.
func runTxGet(args []string) error {
	fs := flag.NewFlagSet("tx get", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	hash := fs.String("hash", "", "transaction hash to query")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *hash == "" {
		return fmt.Errorf("-hash is required")
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := txtypes.NewServiceClient(conn)
	resp, err := client.GetTx(ctx, &txtypes.GetTxRequest{Hash: strings.TrimSpace(*hash)})
	if err != nil {
		return fmt.Errorf("failed to query transaction: %w", err)
	}

	return printJSON(resp)
}

// runTxSearch searches transactions by event query.
func runTxSearch(args []string) error {
	fs := flag.NewFlagSet("tx search", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	query := fs.String("query", "", "Tendermint event query")
	sender := fs.String("sender", "", "sender address (auto-build query)")
	recipient := fs.String("recipient", "", "recipient address (auto-build query)")
	page := fs.Uint64("page", 1, "page number")
	limit := fs.Uint64("limit", 30, "page size")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	eventQuery := strings.TrimSpace(*query)
	if eventQuery == "" {
		eventQuery = buildTxEventQuery(strings.TrimSpace(*sender), strings.TrimSpace(*recipient))
	}
	if eventQuery == "" {
		return fmt.Errorf("either -query or -sender/-recipient is required")
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := txtypes.NewServiceClient(conn)
	resp, err := client.GetTxsEvent(ctx, &txtypes.GetTxsEventRequest{
		Query: eventQuery,
		Page:  *page,
		Limit: *limit,
	})
	if err != nil {
		return fmt.Errorf("failed to search transactions: %w", err)
	}

	result := map[string]any{
		"query":        eventQuery,
		"total":        resp.Total,
		"tx_responses": resp.TxResponses,
	}
	return printJSON(result)
}

// runTxSimulate simulates signed tx bytes to estimate gas.
func runTxSimulate(args []string) error {
	fs := flag.NewFlagSet("tx simulate", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	txBytesB64 := fs.String("tx-bytes-base64", "", "base64-encoded tx bytes")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*txBytesB64) == "" {
		return fmt.Errorf("-tx-bytes-base64 is required")
	}

	txBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(*txBytesB64))
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := txtypes.NewServiceClient(conn)
	resp, err := client.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes})
	if err != nil {
		return fmt.Errorf("failed to simulate transaction: %w", err)
	}

	return printJSON(resp)
}

// runTxBroadcast broadcasts signed tx bytes to the network.
func runTxBroadcast(args []string) error {
	fs := flag.NewFlagSet("tx broadcast", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	txBytesB64 := fs.String("tx-bytes-base64", "", "base64-encoded tx bytes")
	mode := fs.String("mode", "sync", "broadcast mode (async|sync|block)")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*txBytesB64) == "" {
		return fmt.Errorf("-tx-bytes-base64 is required")
	}

	txBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(*txBytesB64))
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	broadcastMode, err := parseBroadcastMode(*mode)
	if err != nil {
		return err
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := txtypes.NewServiceClient(conn)
	resp, err := client.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{TxBytes: txBytes, Mode: broadcastMode})
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return printJSON(resp)
}

// buildTxEventQuery builds a default event query from sender/recipient.
func buildTxEventQuery(sender, recipient string) string {
	parts := make([]string, 0, 2)
	if sender != "" {
		parts = append(parts, fmt.Sprintf("message.sender='%s'", sender))
	}
	if recipient != "" {
		parts = append(parts, fmt.Sprintf("transfer.recipient='%s'", recipient))
	}
	return strings.Join(parts, " AND ")
}

// parseBroadcastMode converts a string mode into protobuf BroadcastMode.
func parseBroadcastMode(mode string) (txtypes.BroadcastMode, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "async":
		return txtypes.BroadcastMode_BROADCAST_MODE_ASYNC, nil
	case "sync":
		return txtypes.BroadcastMode_BROADCAST_MODE_SYNC, nil
	case "block":
		return txtypes.BroadcastMode_BROADCAST_MODE_BLOCK, nil
	default:
		return txtypes.BroadcastMode_BROADCAST_MODE_UNSPECIFIED, fmt.Errorf("unsupported mode: %s", mode)
	}
}
