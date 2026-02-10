package main

import (
	"flag"
	"fmt"
	"time"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// runBalanceCommand routes balance subcommands.
func runBalanceCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("balance subcommand is required (all/denom)")
	}

	switch args[0] {
	case "all":
		return runBalanceAll(args[1:])
	case "denom":
		return runBalanceByDenom(args[1:])
	default:
		return fmt.Errorf("unsupported balance subcommand: %s", args[0])
	}
}

// runBalanceAll queries all balances for an address.
func runBalanceAll(args []string) error {
	fs := flag.NewFlagSet("balance all", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	address := fs.String("address", "", "account address to query")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *address == "" {
		return fmt.Errorf("-address is required")
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := banktypes.NewQueryClient(conn)
	resp, err := client.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{Address: *address})
	if err != nil {
		return fmt.Errorf("failed to query balances: %w", err)
	}

	result := map[string]any{
		"address":  *address,
		"balances": resp.Balances,
	}
	return printJSON(result)
}

// runBalanceByDenom queries a specific denom balance for an address.
func runBalanceByDenom(args []string) error {
	fs := flag.NewFlagSet("balance denom", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	address := fs.String("address", "", "account address to query")
	denom := fs.String("denom", "ulwp", "denom to query")
	timeout := fs.Duration("timeout", 7*time.Second, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *address == "" {
		return fmt.Errorf("-address is required")
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := banktypes.NewQueryClient(conn)
	resp, err := client.Balance(ctx, &banktypes.QueryBalanceRequest{Address: *address, Denom: *denom})
	if err != nil {
		return fmt.Errorf("failed to query denom balance: %w", err)
	}

	result := map[string]any{
		"address": *address,
		"denom":   *denom,
		"balance": resp.Balance,
	}
	return printJSON(result)
}
