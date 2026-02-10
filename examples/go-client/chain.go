package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/LumiWave/lumiwave-protocol/x/lumiwaveprotocol/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// runChainCommand routes chain subcommands.
func runChainCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("chain subcommand is required (status/block/watch/account/module-params)")
	}

	switch args[0] {
	case "status":
		return runChainStatus(args[1:])
	case "block":
		return runChainBlock(args[1:])
	case "watch":
		return runChainWatch(args[1:])
	case "account":
		return runChainAccount(args[1:])
	case "module-params":
		return runModuleParams(args[1:])
	default:
		return fmt.Errorf("unsupported chain subcommand: %s", args[0])
	}
}

// runChainStatus queries node info, sync status, and latest block.
func runChainStatus(args []string) error {
	fs := flag.NewFlagSet("chain status", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := cmtservice.NewServiceClient(conn)
	nodeInfo, err := client.GetNodeInfo(ctx, &cmtservice.GetNodeInfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to query node info: %w", err)
	}

	syncing, err := client.GetSyncing(ctx, &cmtservice.GetSyncingRequest{})
	if err != nil {
		return fmt.Errorf("failed to query sync status: %w", err)
	}

	latestBlock, err := client.GetLatestBlock(ctx, &cmtservice.GetLatestBlockRequest{})
	if err != nil {
		return fmt.Errorf("failed to query latest block: %w", err)
	}

	result := map[string]any{
		"node_info":    nodeInfo.GetDefaultNodeInfo(),
		"application":  nodeInfo.GetApplicationVersion(),
		"is_syncing":   syncing.GetSyncing(),
		"latest_block": latestBlock.GetBlock(),
	}
	return printJSON(result)
}

// runChainBlock queries a block by height or the latest block.
func runChainBlock(args []string) error {
	fs := flag.NewFlagSet("chain block", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	height := fs.Int64("height", 0, "block height (0 means latest block)")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := cmtservice.NewServiceClient(conn)
	if *height <= 0 {
		resp, err := client.GetLatestBlock(ctx, &cmtservice.GetLatestBlockRequest{})
		if err != nil {
			return fmt.Errorf("failed to query latest block: %w", err)
		}
		return printJSON(resp)
	}

	resp, err := client.GetBlockByHeight(ctx, &cmtservice.GetBlockByHeightRequest{Height: *height})
	if err != nil {
		return fmt.Errorf("failed to query block (height=%d): %w", *height, err)
	}
	return printJSON(resp)
}

// runChainAccount queries account information (including account number/sequence use cases).
func runChainAccount(args []string) error {
	fs := flag.NewFlagSet("chain account", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	address := fs.String("address", "", "account address to query")
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

	client := authtypes.NewQueryClient(conn)
	resp, err := client.Account(ctx, &authtypes.QueryAccountRequest{Address: *address})
	if err != nil {
		return fmt.Errorf("failed to query account: %w", err)
	}

	return printJSON(resp)
}

// runModuleParams queries parameters of the custom lumiwaveprotocol module.
func runModuleParams(args []string) error {
	fs := flag.NewFlagSet("chain module-params", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := types.NewQueryClient(conn)
	resp, err := client.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		return fmt.Errorf("failed to query module params: %w", err)
	}

	return printJSON(resp)
}
