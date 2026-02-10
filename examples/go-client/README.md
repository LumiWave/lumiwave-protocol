# LumiWave Go Client Example

`examples/go-client` is a CLI sample that interacts with the LumiWave chain over gRPC.

## Run From Project Root
Run commands from the project root (`lumiwave-protocol`) like this:

```bash
go run ./examples/go-client <command> <subcommand> [flags]
```

The default gRPC endpoint is `localhost:9090`. Most commands support overriding it with `--grpc-endpoint`.

## Command Groups
- `wallet`: Create/restore/validate wallets
- `balance`: Query balances
- `tx`: Send/query/search/simulate/broadcast transactions
- `chain`: Query chain status/blocks/accounts/module params and watch new blocks

## Quick Start

### 1) Create a Wallet
```bash
go run ./examples/go-client wallet create
```

### 2) Query All Balances for an Address
```bash
go run ./examples/go-client balance all --address lumi1...
```

### 3) Send with Mnemonic
```bash
go run ./examples/go-client tx send \
  --from-mnemonic "<MNEMONIC>" \
  --to-address lumi1... \
  --amount 1000 \
  --denom ulwp
```

### 4) Send with Private Key Hex
```bash
go run ./examples/go-client tx send-privkey \
  --from-private-key-hex <HEX> \
  --to-address lumi1... \
  --amount 1000 \
  --denom ulwp
```

### 5) Get a Transaction by Hash
```bash
go run ./examples/go-client tx get --hash ABCDEF...
```

### 6) Watch Latest Blocks
```bash
go run ./examples/go-client chain watch --interval 2s
```

### 7) Query Chain Status
```bash
go run ./examples/go-client chain status
```

## Main Subcommands

### wallet
- `create`: Create a new mnemonic-based wallet
- `from-mnemonic`: Restore a wallet from an existing mnemonic
- `validate`: Validate mnemonic format

Example:
```bash
go run ./examples/go-client wallet from-mnemonic --mnemonic "<MNEMONIC>"
go run ./examples/go-client wallet validate --mnemonic "<MNEMONIC>"
```

### balance
- `all`: Query all balances for an address
- `denom`: Query balance for a specific denom

Example:
```bash
go run ./examples/go-client balance denom --address lumi1... --denom ulwp
```

### tx
- `send`: Send tokens using mnemonic
- `send-privkey`: Send tokens using private key hex
- `get`: Query one transaction by hash
- `search`: Search transactions by event query
- `simulate`: Simulate base64 tx bytes
- `broadcast`: Broadcast base64 tx bytes

Example:
```bash
go run ./examples/go-client tx search --sender lumi1...
go run ./examples/go-client tx broadcast --tx-bytes-base64 "<BASE64_TX_BYTES>" --mode sync
```

### chain
- `status`: Query node/sync/latest block info
- `block`: Query a specific height or latest block
- `watch`: Poll and print newly detected blocks
- `account`: Query account details
- `module-params`: Query `lumiwaveprotocol` module params

Example:
```bash
go run ./examples/go-client chain block --height 123
go run ./examples/go-client chain account --address lumi1...
go run ./examples/go-client chain module-params
```

## Tests
Run tests for `examples/go-client`:

```bash
go test ./examples/go-client/...
```
