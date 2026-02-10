package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printRootUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "wallet":
		err = runWalletCommand(os.Args[2:])
	case "balance":
		err = runBalanceCommand(os.Args[2:])
	case "tx":
		err = runTxCommand(os.Args[2:])
	case "chain":
		err = runChainCommand(os.Args[2:])
	default:
		printRootUsage()
		err = fmt.Errorf("unsupported command: %s", os.Args[1])
	}

	if err != nil {
		exitErr(err)
	}
}

// printRootUsage prints the root usage instructions.
func printRootUsage() {
	fmt.Println(`LumiWave Go Client Sample

Usage:
  go run ./examples/go-client <command> <subcommand> [flags]

Commands:
  wallet   Wallet create/restore/validate
  balance  Coin/token balance queries
  tx       Transaction query/search/simulate/broadcast
  chain    Chain status/block/account/module params

wallet subcommands:
  create          Create a new mnemonic wallet
  from-mnemonic   Restore wallet from mnemonic
  validate        Validate mnemonic

balance subcommands:
  all             Query all balances for an address
  denom           Query a specific denom balance

tx subcommands:
  send            Send using mnemonic
  send-privkey    Send using private key(hex)
  get             Get one tx by hash
  search          Search tx list by event query
  simulate        Simulate tx bytes(base64)
  broadcast       Broadcast tx bytes(base64)

chain subcommands:
  status          Query node status/sync/latest block
  block           Query a specific height or latest block
  watch           Watch latest blocks and print each new block snapshot
  account         Query account info(account number/sequence)
  module-params   Query lumiwaveprotocol module params

Examples:
  go run ./examples/go-client wallet create
  go run ./examples/go-client balance all --address lumi1...
  go run ./examples/go-client tx send --from-mnemonic "..." --to-address lumi1... --amount 1000 --denom ulwp
  go run ./examples/go-client tx send-privkey --from-private-key-hex <HEX> --to-address lumi1... --amount 1000 --denom ulwp
  go run ./examples/go-client tx get --hash ABCDEF...
  go run ./examples/go-client chain watch --interval 2s
  go run ./examples/go-client chain status`)
}
