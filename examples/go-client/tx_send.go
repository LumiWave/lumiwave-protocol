package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	bip39 "github.com/cosmos/go-bip39"
	"google.golang.org/grpc"
)

// runTxSend builds/signs/broadcasts a transfer tx from mnemonic (ulwp or custom denom).
func runTxSend(args []string) error {
	fs := flag.NewFlagSet("tx send", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	chainID := fs.String("chain-id", "", "chain ID (auto-detect if empty)")
	mnemonic := fs.String("from-mnemonic", "", "sender wallet mnemonic")
	toAddress := fs.String("to-address", "", "recipient wallet address")
	amount := fs.String("amount", "1000", "send amount (integer string)")
	denom := fs.String("denom", "ulwp", "send denom")
	feeAmount := fs.String("fee-amount", "500", "fee amount (integer string)")
	feeDenom := fs.String("fee-denom", "ulwp", "fee denom")
	gasLimit := fs.Uint64("gas-limit", 200000, "gas limit")
	memo := fs.String("memo", "", "transaction memo")
	mode := fs.String("mode", "sync", "broadcast mode (async|sync|block)")
	simulateOnly := fs.Bool("simulate-only", false, "if true, run simulation only without broadcasting")
	coinType := fs.Uint("coin-type", 118, "BIP44 coin type")
	account := fs.Uint("account", 0, "BIP44 account")
	index := fs.Uint("index", 0, "BIP44 address index")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*mnemonic) == "" {
		return fmt.Errorf("-from-mnemonic is required")
	}
	if strings.TrimSpace(*toAddress) == "" {
		return fmt.Errorf("-to-address is required")
	}

	// Derive private key and sender address from mnemonic.
	privKey, fromAddr, err := derivePrivKeyAndAddress(strings.TrimSpace(*mnemonic), uint32(*coinType), uint32(*account), uint32(*index))
	if err != nil {
		return err
	}

	toAddr, err := sdk.AccAddressFromBech32(strings.TrimSpace(*toAddress))
	if err != nil {
		return fmt.Errorf("failed to parse recipient address: %w", err)
	}

	sendCoin, err := buildCoin(*denom, *amount)
	if err != nil {
		return fmt.Errorf("failed to parse send coin: %w", err)
	}

	feeCoin, err := buildCoin(*feeDenom, *feeAmount)
	if err != nil {
		return fmt.Errorf("failed to parse fee coin: %w", err)
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Auto-detect chain-id from node info when not provided.
	resolvedChainID := strings.TrimSpace(*chainID)
	if resolvedChainID == "" {
		resolvedChainID, err = detectChainID(ctx, conn)
		if err != nil {
			return err
		}
	}

	// Query account number/sequence for signing.
	baseAcc, err := queryBaseAccount(ctx, conn, fromAddr.String())
	if err != nil {
		return err
	}

	// Build MsgSend.
	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sendCoin))

	// Build and sign tx bytes using SIGN_MODE_DIRECT.
	txBytes, err := buildAndSignMsgSendTx(msg, privKey, resolvedChainID, baseAcc.AccountNumber, baseAcc.Sequence, sdk.NewCoins(feeCoin), *gasLimit, *memo)
	if err != nil {
		return err
	}

	serviceClient := txtypes.NewServiceClient(conn)

	// Simulate only when requested.
	if *simulateOnly {
		simResp, err := serviceClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes})
		if err != nil {
			return fmt.Errorf("simulation failed: %w", err)
		}
		result := map[string]any{
			"from":            fromAddr.String(),
			"to":              toAddr.String(),
			"amount":          sendCoin.String(),
			"chain_id":        resolvedChainID,
			"account_number":  baseAcc.AccountNumber,
			"sequence":        baseAcc.Sequence,
			"tx_bytes_base64": base64.StdEncoding.EncodeToString(txBytes),
			"simulate":        simResp,
		}
		return printJSON(result)
	}

	broadcastMode, err := parseBroadcastMode(*mode)
	if err != nil {
		return err
	}

	// Broadcast signed tx bytes.
	broadcastResp, err := serviceClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    broadcastMode,
	})
	if err != nil {
		return fmt.Errorf("broadcast failed: %w", err)
	}

	result := map[string]any{
		"from":            fromAddr.String(),
		"to":              toAddr.String(),
		"amount":          sendCoin.String(),
		"fee":             feeCoin.String(),
		"chain_id":        resolvedChainID,
		"account_number":  baseAcc.AccountNumber,
		"sequence":        baseAcc.Sequence,
		"tx_bytes_base64": base64.StdEncoding.EncodeToString(txBytes),
		"broadcast":       broadcastResp,
	}
	return printJSON(result)
}

// runTxSendWithPrivateKey sends ulwp (or custom denom) using private key hex.
func runTxSendWithPrivateKey(args []string) error {
	fs := flag.NewFlagSet("tx send-privkey", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	chainID := fs.String("chain-id", "", "chain ID (auto-detect if empty)")
	privateKeyHex := fs.String("from-private-key-hex", "", "sender private key (hex)")
	toAddress := fs.String("to-address", "", "recipient wallet address")
	amount := fs.String("amount", "1000", "send amount (integer string)")
	denom := fs.String("denom", "ulwp", "send denom")
	feeAmount := fs.String("fee-amount", "500", "fee amount (integer string)")
	feeDenom := fs.String("fee-denom", "ulwp", "fee denom")
	gasLimit := fs.Uint64("gas-limit", 200000, "gas limit")
	memo := fs.String("memo", "", "transaction memo")
	mode := fs.String("mode", "sync", "broadcast mode (async|sync|block)")
	simulateOnly := fs.Bool("simulate-only", false, "if true, run simulation only without broadcasting")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*privateKeyHex) == "" {
		return fmt.Errorf("-from-private-key-hex is required")
	}
	if strings.TrimSpace(*toAddress) == "" {
		return fmt.Errorf("-to-address is required")
	}

	// Restore private key and sender address from hex.
	privKey, fromAddr, err := parsePrivKeyAndAddress(strings.TrimSpace(*privateKeyHex))
	if err != nil {
		return err
	}

	toAddr, err := sdk.AccAddressFromBech32(strings.TrimSpace(*toAddress))
	if err != nil {
		return fmt.Errorf("failed to parse recipient address: %w", err)
	}

	sendCoin, err := buildCoin(*denom, *amount)
	if err != nil {
		return fmt.Errorf("failed to parse send coin: %w", err)
	}

	feeCoin, err := buildCoin(*feeDenom, *feeAmount)
	if err != nil {
		return fmt.Errorf("failed to parse fee coin: %w", err)
	}

	ctx, cancel := newTimeoutContext(*timeout)
	defer cancel()

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	resolvedChainID := strings.TrimSpace(*chainID)
	if resolvedChainID == "" {
		resolvedChainID, err = detectChainID(ctx, conn)
		if err != nil {
			return err
		}
	}

	baseAcc, err := queryBaseAccount(ctx, conn, fromAddr.String())
	if err != nil {
		return err
	}

	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sendCoin))
	txBytes, err := buildAndSignMsgSendTx(msg, privKey, resolvedChainID, baseAcc.AccountNumber, baseAcc.Sequence, sdk.NewCoins(feeCoin), *gasLimit, *memo)
	if err != nil {
		return err
	}

	serviceClient := txtypes.NewServiceClient(conn)
	if *simulateOnly {
		simResp, err := serviceClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes})
		if err != nil {
			return fmt.Errorf("simulation failed: %w", err)
		}
		result := map[string]any{
			"from":            fromAddr.String(),
			"to":              toAddr.String(),
			"amount":          sendCoin.String(),
			"chain_id":        resolvedChainID,
			"account_number":  baseAcc.AccountNumber,
			"sequence":        baseAcc.Sequence,
			"tx_bytes_base64": base64.StdEncoding.EncodeToString(txBytes),
			"simulate":        simResp,
		}
		return printJSON(result)
	}

	broadcastMode, err := parseBroadcastMode(*mode)
	if err != nil {
		return err
	}

	broadcastResp, err := serviceClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    broadcastMode,
	})
	if err != nil {
		return fmt.Errorf("broadcast failed: %w", err)
	}

	result := map[string]any{
		"from":            fromAddr.String(),
		"to":              toAddr.String(),
		"amount":          sendCoin.String(),
		"fee":             feeCoin.String(),
		"chain_id":        resolvedChainID,
		"account_number":  baseAcc.AccountNumber,
		"sequence":        baseAcc.Sequence,
		"tx_bytes_base64": base64.StdEncoding.EncodeToString(txBytes),
		"broadcast":       broadcastResp,
	}
	return printJSON(result)
}

// derivePrivKeyAndAddress derives secp256k1 private key and address from mnemonic+HD path.
func derivePrivKeyAndAddress(mnemonic string, coinType, account, index uint32) (*secp256k1.PrivKey, sdk.AccAddress, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, nil, fmt.Errorf("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, "")
	masterPriv, chainCode := hd.ComputeMastersFromSeed(seed)
	hdPath := hd.CreateHDPath(coinType, account, index).String()

	derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, chainCode, hdPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive HD private key: %w", err)
	}

	privKey := &secp256k1.PrivKey{Key: derivedPriv}
	addr := sdk.AccAddress(privKey.PubKey().Address())
	return privKey, addr, nil
}

// parsePrivKeyAndAddress restores secp256k1 private key and address from hex string.
func parsePrivKeyAndAddress(privateKeyHex string) (*secp256k1.PrivKey, sdk.AccAddress, error) {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(privateKeyHex)), "0x")
	keyBytes, err := hex.DecodeString(normalized)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode private key hex: %w", err)
	}
	if len(keyBytes) != secp256k1.PrivKeySize {
		return nil, nil, fmt.Errorf("invalid private key length: %d", len(keyBytes))
	}

	privKey := &secp256k1.PrivKey{Key: keyBytes}
	addr := sdk.AccAddress(privKey.PubKey().Address())
	return privKey, addr, nil
}

// buildCoin builds a Coin from denom and integer amount string.
func buildCoin(denom, amount string) (sdk.Coin, error) {
	intAmount, ok := sdkmath.NewIntFromString(strings.TrimSpace(amount))
	if !ok || !intAmount.IsPositive() {
		return sdk.Coin{}, fmt.Errorf("amount must be a positive integer: %s", amount)
	}
	return sdk.NewCoin(strings.TrimSpace(denom), intAmount), nil
}

// detectChainID queries chain-id from node gRPC info.
func detectChainID(ctx context.Context, conn *grpc.ClientConn) (string, error) {
	cmtClient := cmtservice.NewServiceClient(conn)
	resp, err := cmtClient.GetNodeInfo(ctx, &cmtservice.GetNodeInfoRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to auto-detect chain-id: %w", err)
	}

	chainID := strings.TrimSpace(resp.GetDefaultNodeInfo().GetNetwork())
	if chainID == "" {
		return "", fmt.Errorf("chain-id is empty in node info")
	}
	return chainID, nil
}

// queryBaseAccount fetches base account info required for signing.
func queryBaseAccount(ctx context.Context, conn *grpc.ClientConn, address string) (*authtypes.BaseAccount, error) {
	authClient := authtypes.NewQueryClient(conn)
	resp, err := authClient.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: address})
	if err != nil {
		return nil, fmt.Errorf("failed to query account info: %w", err)
	}
	if resp.GetInfo() == nil {
		return nil, fmt.Errorf("account info is empty: %s", address)
	}
	return resp.GetInfo(), nil
}

// buildAndSignMsgSendTx builds and signs MsgSend tx bytes with SIGN_MODE_DIRECT.
func buildAndSignMsgSendTx(
	msg *banktypes.MsgSend,
	privKey *secp256k1.PrivKey,
	chainID string,
	accountNumber uint64,
	sequence uint64,
	fees sdk.Coins,
	gasLimit uint64,
	memo string,
) ([]byte, error) {
	msgAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message to Any: %w", err)
	}

	txBody := &txtypes.TxBody{
		Messages: []*codectypes.Any{msgAny},
		Memo:     memo,
	}
	bodyBytes, err := txBody.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx body: %w", err)
	}

	pubKeyAny, err := codectypes.NewAnyWithValue(privKey.PubKey())
	if err != nil {
		return nil, fmt.Errorf("failed to convert public key to Any: %w", err)
	}

	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{
			{
				PublicKey: pubKeyAny,
				ModeInfo: &txtypes.ModeInfo{
					Sum: &txtypes.ModeInfo_Single_{
						Single: &txtypes.ModeInfo_Single{
							Mode: signingtypes.SignMode_SIGN_MODE_DIRECT,
						},
					},
				},
				Sequence: sequence,
			},
		},
		Fee: &txtypes.Fee{
			Amount:   fees,
			GasLimit: gasLimit,
		},
	}
	authInfoBytes, err := authInfo.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to encode auth info: %w", err)
	}

	signBytes, err := authtx.DirectSignBytes(bodyBytes, authInfoBytes, chainID, accountNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to build sign bytes: %w", err)
	}

	sig, err := privKey.Sign(signBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	txRaw := &txtypes.TxRaw{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		Signatures:    [][]byte{sig},
	}
	txBytes, err := txRaw.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx raw: %w", err)
	}

	return txBytes, nil
}
