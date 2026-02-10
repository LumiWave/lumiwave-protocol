package main

import (
	"encoding/hex"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func TestBuildTxEventQuery(t *testing.T) {
	t.Parallel()

	if got := buildTxEventQuery("", ""); got != "" {
		t.Fatalf("expected empty query, got %q", got)
	}

	if got := buildTxEventQuery("lumi1sender", ""); got != "message.sender='lumi1sender'" {
		t.Fatalf("unexpected sender query: %q", got)
	}

	want := "message.sender='lumi1sender' AND transfer.recipient='lumi1recipient'"
	if got := buildTxEventQuery("lumi1sender", "lumi1recipient"); got != want {
		t.Fatalf("unexpected combined query: %q", got)
	}
}

func TestParseBroadcastMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want txtypes.BroadcastMode
		ok   bool
	}{
		{"async", txtypes.BroadcastMode_BROADCAST_MODE_ASYNC, true},
		{"SYNC", txtypes.BroadcastMode_BROADCAST_MODE_SYNC, true},
		{" block ", txtypes.BroadcastMode_BROADCAST_MODE_BLOCK, true},
		{"invalid", txtypes.BroadcastMode_BROADCAST_MODE_UNSPECIFIED, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseBroadcastMode(tc.in)
			if tc.ok {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Fatalf("expected %v, got %v", tc.want, got)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBuildCoin(t *testing.T) {
	t.Parallel()

	coin, err := buildCoin("ulwp", "1000")
	if err != nil {
		t.Fatalf("buildCoin failed: %v", err)
	}
	if coin.Denom != "ulwp" || coin.Amount.String() != "1000" {
		t.Fatalf("unexpected coin: %s", coin.String())
	}

	if _, err := buildCoin("ulwp", "0"); err == nil {
		t.Fatal("expected error for zero amount")
	}
	if _, err := buildCoin("ulwp", "-1"); err == nil {
		t.Fatal("expected error for negative amount")
	}
	if _, err := buildCoin("ulwp", "abc"); err == nil {
		t.Fatal("expected error for invalid amount")
	}
}

func TestParsePrivKeyAndAddress(t *testing.T) {
	t.Parallel()

	priv, addr, err := derivePrivKeyAndAddress(testMnemonic, 118, 0, 0)
	if err != nil {
		t.Fatalf("derivePrivKeyAndAddress failed: %v", err)
	}

	hexKey := hex.EncodeToString(priv.Bytes())
	parsedPriv, parsedAddr, err := parsePrivKeyAndAddress(hexKey)
	if err != nil {
		t.Fatalf("parsePrivKeyAndAddress failed: %v", err)
	}

	if parsedAddr.String() != addr.String() {
		t.Fatalf("address mismatch: %s vs %s", parsedAddr, addr)
	}
	if got := hex.EncodeToString(parsedPriv.Bytes()); got != hexKey {
		t.Fatalf("private key mismatch: %s vs %s", got, hexKey)
	}

	_, _, err = parsePrivKeyAndAddress("0x" + hexKey)
	if err != nil {
		t.Fatalf("expected 0x-prefixed hex to work: %v", err)
	}

	if _, _, err := parsePrivKeyAndAddress("zzzz"); err == nil {
		t.Fatal("expected error for invalid hex")
	}
	if _, _, err := parsePrivKeyAndAddress("abcd"); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestBuildAndSignMsgSendTx(t *testing.T) {
	t.Parallel()

	priv, fromAddr, err := derivePrivKeyAndAddress(testMnemonic, 118, 0, 0)
	if err != nil {
		t.Fatalf("derivePrivKeyAndAddress failed: %v", err)
	}

	_, toAddr, err := derivePrivKeyAndAddress(testMnemonic, 118, 0, 1)
	if err != nil {
		t.Fatalf("toAddr derive failed: %v", err)
	}

	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewInt64Coin("ulwp", 1000)))
	txBytes, err := buildAndSignMsgSendTx(msg, priv, "lumiwaveprotocol", 1, 2, sdk.NewCoins(sdk.NewInt64Coin("ulwp", 500)), 200000, "memo")
	if err != nil {
		t.Fatalf("buildAndSignMsgSendTx failed: %v", err)
	}
	if len(txBytes) == 0 {
		t.Fatal("tx bytes should not be empty")
	}

	var raw txtypes.TxRaw
	if err := raw.Unmarshal(txBytes); err != nil {
		t.Fatalf("unmarshal tx raw failed: %v", err)
	}
	if len(raw.Signatures) != 1 {
		t.Fatalf("expected one signature, got %d", len(raw.Signatures))
	}
	if len(raw.BodyBytes) == 0 || len(raw.AuthInfoBytes) == 0 {
		t.Fatal("body/auth bytes should not be empty")
	}
}

func TestRunTxCommandValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"missing subcommand", runTxCommand(nil)},
		{"unknown subcommand", runTxCommand([]string{"unknown"})},
		{"send missing required", runTxSend(nil)},
		{"send-privkey missing required", runTxSendWithPrivateKey(nil)},
		{"get missing hash", runTxGet(nil)},
		{"search missing query", runTxSearch(nil)},
		{"simulate missing bytes", runTxSimulate(nil)},
		{"broadcast missing bytes", runTxBroadcast(nil)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}

	_, err := parseBroadcastMode("bad-mode")
	if err == nil || !strings.Contains(err.Error(), "unsupported mode") {
		t.Fatalf("unexpected parseBroadcastMode error: %v", err)
	}
}
