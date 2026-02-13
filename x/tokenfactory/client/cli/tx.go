package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"

	"github.com/LumiWave/lumiwave-protocol/x/tokenfactory/types"
)

const (
	FlagMintToAddress = "mint-to-address"
	FlagBurnFromAddr  = "burn-from-address"
	FlagMetadataFile  = "metadata-file"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokenfactory transaction subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		NewCreateDenomCmd(),
		NewMintCmd(),
		NewBurnCmd(),
		NewChangeAdminCmd(),
		NewSetBeforeSendHookCmd(),
		NewSetDenomMetadataCmd(),
	)

	return cmd
}

func NewCreateDenomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-denom [subdenom]",
		Short: "Create a new factory denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateDenom(clientCtx.GetFromAddress().String(), args[0])
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewMintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [amount]",
		Short: "Mint tokens for a factory denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgMint(clientCtx.GetFromAddress().String(), amount)
			mintToAddress, err := cmd.Flags().GetString(FlagMintToAddress)
			if err != nil {
				return err
			}
			if mintToAddress != "" {
				msg.MintToAddress = mintToAddress
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMintToAddress, "", "optional bech32 address to receive minted tokens (defaults to sender)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewBurnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn [amount]",
		Short: "Burn tokens for a factory denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgBurn(clientCtx.GetFromAddress().String(), amount)
			burnFromAddress, err := cmd.Flags().GetString(FlagBurnFromAddr)
			if err != nil {
				return err
			}
			if burnFromAddress != "" {
				msg.BurnFromAddress = burnFromAddress
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagBurnFromAddr, "", "optional bech32 address to burn from (defaults to sender)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewChangeAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-admin [denom] [new-admin]",
		Short: "Change admin for a factory denom",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgChangeAdmin(clientCtx.GetFromAddress().String(), args[0], args[1])
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSetBeforeSendHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-beforesend-hook [denom] [cosmwasm-address]",
		Short: "Set a CosmWasm before-send hook address for a factory denom",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetBeforeSendHook(clientCtx.GetFromAddress().String(), args[0], args[1])
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSetDenomMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-denom-metadata",
		Short: "Set metadata for a factory denom",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			metadataFile, err := cmd.Flags().GetString(FlagMetadataFile)
			if err != nil {
				return err
			}
			if metadataFile == "" {
				return fmt.Errorf("--%s is required", FlagMetadataFile)
			}

			bz, err := os.ReadFile(metadataFile)
			if err != nil {
				return err
			}

			var metadata banktypes.Metadata
			if err := json.Unmarshal(bz, &metadata); err != nil {
				return err
			}

			msg := types.NewMsgSetDenomMetadata(clientCtx.GetFromAddress().String(), metadata)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMetadataFile, "", "path to denom metadata JSON file")
	_ = cmd.MarkFlagRequired(FlagMetadataFile)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
