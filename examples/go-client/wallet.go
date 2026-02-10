package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bip39 "github.com/cosmos/go-bip39"
)

// WalletInfo contains sample wallet information.
type WalletInfo struct {
	Mnemonic      string `json:"mnemonic,omitempty"`
	HDPath        string `json:"hd_path"`
	Address       string `json:"address"`
	PublicKeyHex  string `json:"public_key_hex"`
	PrivateKeyHex string `json:"private_key_hex"`
}

// runWalletCommand routes wallet subcommands.
func runWalletCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("wallet subcommand is required (create/from-mnemonic/validate)")
	}

	switch args[0] {
	case "create":
		return runWalletCreate(args[1:])
	case "from-mnemonic":
		return runWalletFromMnemonic(args[1:])
	case "validate":
		return runWalletValidate(args[1:])
	default:
		return fmt.Errorf("unsupported wallet subcommand: %s", args[0])
	}
}

// runWalletCreate generates a new mnemonic and prints wallet info.
func runWalletCreate(args []string) error {
	fs := flag.NewFlagSet("wallet create", flag.ContinueOnError)
	words := fs.Int("words", 24, "mnemonic word count (12,15,18,21,24)")
	coinType := fs.Uint("coin-type", 118, "BIP44 coin type")
	account := fs.Uint("account", 0, "BIP44 account")
	index := fs.Uint("index", 0, "BIP44 address index")
	if err := fs.Parse(args); err != nil {
		return err
	}

	entropyBits, err := wordsToEntropyBits(*words)
	if err != nil {
		return err
	}

	entropy, err := bip39.NewEntropy(entropyBits)
	if err != nil {
		return fmt.Errorf("failed to generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	wallet, err := deriveWalletFromMnemonic(mnemonic, uint32(*coinType), uint32(*account), uint32(*index), true)
	if err != nil {
		return err
	}

	return printJSON(wallet)
}

// runWalletFromMnemonic restores wallet info from an existing mnemonic.
func runWalletFromMnemonic(args []string) error {
	fs := flag.NewFlagSet("wallet from-mnemonic", flag.ContinueOnError)
	mnemonic := fs.String("mnemonic", "", "mnemonic phrase to restore")
	coinType := fs.Uint("coin-type", 118, "BIP44 coin type")
	account := fs.Uint("account", 0, "BIP44 account")
	index := fs.Uint("index", 0, "BIP44 address index")
	showMnemonic := fs.Bool("show-mnemonic", false, "include mnemonic in output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*mnemonic) == "" {
		return fmt.Errorf("-mnemonic is required")
	}

	wallet, err := deriveWalletFromMnemonic(*mnemonic, uint32(*coinType), uint32(*account), uint32(*index), *showMnemonic)
	if err != nil {
		return err
	}

	return printJSON(wallet)
}

// runWalletValidate validates a mnemonic.
func runWalletValidate(args []string) error {
	fs := flag.NewFlagSet("wallet validate", flag.ContinueOnError)
	mnemonic := fs.String("mnemonic", "", "mnemonic phrase to validate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*mnemonic) == "" {
		return fmt.Errorf("-mnemonic is required")
	}

	result := map[string]any{
		"valid": bip39.IsMnemonicValid(strings.TrimSpace(*mnemonic)),
	}
	return printJSON(result)
}

// deriveWalletFromMnemonic derives private key/public key/address from mnemonic.
func deriveWalletFromMnemonic(mnemonic string, coinType, account, index uint32, showMnemonic bool) (*WalletInfo, error) {
	if !bip39.IsMnemonicValid(strings.TrimSpace(mnemonic)) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	seed := bip39.NewSeed(strings.TrimSpace(mnemonic), "")
	masterPriv, chainCode := hd.ComputeMastersFromSeed(seed)
	hdPath := hd.CreateHDPath(coinType, account, index).String()

	derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, chainCode, hdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to derive private key from HD path: %w", err)
	}

	priv := &secp256k1.PrivKey{Key: derivedPriv}
	pub := priv.PubKey()
	addr := sdk.AccAddress(pub.Address()).String()

	info := &WalletInfo{
		HDPath:        hdPath,
		Address:       addr,
		PublicKeyHex:  hex.EncodeToString(pub.Bytes()),
		PrivateKeyHex: hex.EncodeToString(priv.Bytes()),
	}
	if showMnemonic {
		info.Mnemonic = strings.TrimSpace(mnemonic)
	}
	return info, nil
}

// wordsToEntropyBits converts mnemonic word count to entropy bits.
func wordsToEntropyBits(words int) (int, error) {
	switch words {
	case 12:
		return 128, nil
	case 15:
		return 160, nil
	case 18:
		return 192, nil
	case 21:
		return 224, nil
	case 24:
		return 256, nil
	default:
		return 0, fmt.Errorf("unsupported word count: %d", words)
	}
}
