package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestWordsToEntropyBits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		words int
		bits  int
		ok    bool
	}{
		{12, 128, true},
		{15, 160, true},
		{18, 192, true},
		{21, 224, true},
		{24, 256, true},
		{11, 0, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("words", func(t *testing.T) {
			bits, err := wordsToEntropyBits(tc.words)
			if tc.ok {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if bits != tc.bits {
					t.Fatalf("expected %d bits, got %d", tc.bits, bits)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for words=%d", tc.words)
			}
		})
	}
}

func TestDeriveWalletFromMnemonic(t *testing.T) {
	t.Parallel()

	wallet, err := deriveWalletFromMnemonic(testMnemonic, 118, 0, 0, true)
	if err != nil {
		t.Fatalf("deriveWalletFromMnemonic failed: %v", err)
	}

	if wallet.Address == "" {
		t.Fatal("address should not be empty")
	}
	if !strings.HasPrefix(wallet.Address, "lumi") {
		t.Fatalf("expected lumi prefix, got %s", wallet.Address)
	}
	if wallet.Mnemonic == "" {
		t.Fatal("mnemonic should be present when showMnemonic=true")
	}
	if len(wallet.PrivateKeyHex) != 64 {
		t.Fatalf("unexpected private key hex length: %d", len(wallet.PrivateKeyHex))
	}
	if len(wallet.PublicKeyHex) == 0 {
		t.Fatal("public key hex should not be empty")
	}

	walletNoMnemonic, err := deriveWalletFromMnemonic(testMnemonic, 118, 0, 0, false)
	if err != nil {
		t.Fatalf("deriveWalletFromMnemonic failed: %v", err)
	}
	if walletNoMnemonic.Mnemonic != "" {
		t.Fatal("mnemonic should be empty when showMnemonic=false")
	}
}

func TestDeriveWalletFromMnemonic_InvalidMnemonic(t *testing.T) {
	t.Parallel()

	_, err := deriveWalletFromMnemonic("invalid mnemonic", 118, 0, 0, true)
	if err == nil {
		t.Fatal("expected error for invalid mnemonic")
	}
}

func TestRunWalletCommandValidation(t *testing.T) {
	t.Parallel()

	if err := runWalletCommand(nil); err == nil {
		t.Fatal("expected error for missing wallet subcommand")
	}
	if err := runWalletCommand([]string{"unknown"}); err == nil {
		t.Fatal("expected error for unknown wallet subcommand")
	}
	if err := runWalletFromMnemonic(nil); err == nil {
		t.Fatal("expected error when mnemonic is missing")
	}
	if err := runWalletValidate(nil); err == nil {
		t.Fatal("expected error when mnemonic is missing")
	}
	if err := runWalletCreate([]string{"-words", "13"}); err == nil {
		t.Fatal("expected error for unsupported words")
	}
}

func TestRunWalletValidate_ValidMnemonic(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	if err := runWalletValidate([]string{"-mnemonic", testMnemonic}); err != nil {
		t.Fatalf("runWalletValidate failed: %v", err)
	}

	_ = w.Close()
	outBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("stdout read failed: %v", err)
	}
	out := string(outBytes)
	if !strings.Contains(out, "\"valid\": true") {
		t.Fatalf("unexpected output: %s", out)
	}
}
