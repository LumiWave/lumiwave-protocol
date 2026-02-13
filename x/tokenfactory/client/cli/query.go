package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokenfactory query subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdParams(),
		GetCmdDenomAuthorityMetadata(),
		GetCmdDenomsFromCreator(),
		GetCmdAllBeforeSendHooks(),
		GetCmdDenomBeforeSendHook(),
	)

	return cmd
}

func GetCmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the tokenfactory parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdDenomAuthorityMetadata() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-authority-metadata [denom]",
		Short: "Get authority metadata for a denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DenomAuthorityMetadata(cmd.Context(), &types.QueryDenomAuthorityMetadataRequest{Denom: args[0]})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdDenomsFromCreator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denoms-from-creator [creator]",
		Short: "Get all denoms created by an address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DenomsFromCreator(cmd.Context(), &types.QueryDenomsFromCreatorRequest{Creator: args[0]})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdAllBeforeSendHooks() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-before-send-hooks",
		Short: "List all registered before-send hooks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.AllBeforeSendHooksAddresses(cmd.Context(), &types.QueryAllBeforeSendHooksAddressesRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdDenomBeforeSendHook() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-before-send-hook [denom]",
		Short: "Get the before-send hook for a denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.BeforeSendHookAddress(cmd.Context(), &types.QueryBeforeSendHookAddressRequest{Denom: args[0]})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
