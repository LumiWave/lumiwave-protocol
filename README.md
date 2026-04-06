# LumiWave Protocol

LumiWave Protocol is a Layer 1 blockchain node implementation built on the Cosmos SDK.  
It is scaffolded using Ignite CLI and designed for extensible custom modules and testnet operations.

- Binary: lumiwave-protocold
- Branch: testnet

---

## Table of Contents

- Overview
- Repository Structure
- Prerequisites
- Quick Start (Local Dev)
- Build
- Run a Local Node (Manual)
- Testnet
- TokenFactory
- Cosmovisor Deployment
- Releases
- Docs
- Contributing
- License

---

## Overview

This repository contains the source code required to build and run the LumiWave Protocol blockchain node.

Main components include:

- Cosmos SDK application wiring (`app/`)
- Node binary entrypoint (`cmd/lumiwave-protocold/`)
- Custom modules (`x/lumiwaveprotocol/`)
- Protobuf definitions (`proto/`)
- Testnet genesis files (`genesis/testnet/`)
- Cosmovisor deployment script (`all_new_deploy_for_cosmovisor.sh`)

---

## Repository Structure

    .
    ├── app/                       # Cosmos SDK app wiring
    ├── cmd/lumiwave-protocold/    # Node binary entrypoint
    ├── docs/                      # Documentation (WIP)
    ├── genesis/testnet/           # Testnet genesis files
    ├── proto/                     # Protobuf definitions
    ├── testutil/sample/           # Test utilities
    ├── x/lumiwaveprotocol/        # Custom module(s)
    ├── Makefile                   # Build helpers
    ├── config.yml                 # Ignite configuration
    └── all_new_deploy_for_cosmovisor.sh

---

## Prerequisites

- Go (version defined in `go.mod`)
- make
- Ignite CLI (optional)
- Cosmovisor (for production environments)

---

## Quick Start (Local Dev)

Run a local development chain using Ignite CLI:

    ignite chain serve --config config_develop_only.yml

This command handles building, initialization, and node startup in a single step.

---

## Build

Build the binary:

    go build -o lumiwave-protocold ./cmd/lumiwave-protocold

You can install the binary using one of the following methods:

### 1) Go Standard Installation

    go install ./cmd/lumiwave-protocold

### 2) Makefile-based Installation (Recommended)

    make install

### 3) Ignite CLI Build and Install

    ignite chain build

---

## Run a Local Node (Manual)

Steps to run a local node manually:

1) Initialize the node

    lumiwave-protocold init localnode --chain-id <CHAIN_ID>

2) Create a key

    lumiwave-protocold keys add alice --keyring-backend test

    lumiwave-protocold keys show alice -a --keyring-backend test

3) Add a genesis account

    lumiwave-protocold add-genesis-account <ADDRESS> 100000000ulwp

4) Generate a gentx

    lumiwave-protocold gentx alice 1000000ulwp \
      --chain-id <CHAIN_ID> \
      --keyring-backend test

5) Collect gentxs

    lumiwave-protocold collect-gentxs

6) Start the node

    lumiwave-protocold start

---

## Testnet

Testnet-related resources are located in the `genesis/testnet/` directory.

### Network Parameters

- chain-id: lumiwaveprotocol
- base denom: ulwp
- display denom: LWP
- decimals: 6

### Public Endpoints

Update the following according to the running environment.

#### Example

Below is an example of publicly exposed endpoints for a running testnet:

- RPC: `https://rpc.testnet.example.io:443`
- REST (LCD): `https://api.testnet.example.io:443`
- gRPC: `grpc.testnet.example.io:443`
- gRPC-web: `https://grpc-web.testnet.example.io:443`
- Explorer: `https://explorer.testnet.example.io/ping-pub`

#### LumiWave Testnet Links

- Dashboard: [https://lwp-testnet-dashboard.lumiwavelab.com](https://lwp-testnet-dashboard.lumiwavelab.com)
- Explorer: [https://lwp-testnet-explorer.lumiwavelab.com](https://lwp-testnet-explorer.lumiwavelab.com)

#### Description

- **RPC**  
  Tendermint RPC endpoint used by CLI tools, wallets, relayers, and monitoring services.

- **REST (LCD)**  
  Cosmos SDK REST API endpoint for querying chain data over HTTP.

- **gRPC**  
  Native gRPC endpoint, recommended for backend services and indexers.

- **gRPC-web**  
  gRPC-web endpoint designed for browser-based applications.

- **Explorer**  
  Block explorer endpoint powered by Ping.Pub.

### Go Client Sample

Go client usage/documentation has been moved to:

    [examples/go-client/README.md](examples/go-client/README.md)

### Peers

Peer configuration is defined in the node configuration file:

    config/config.toml

Update the following fields:

    seeds = ""
    persistent_peers = ""

#### Example

    seeds = "abcd1234efgh5678@seed1.example.com:26656"

    persistent_peers = "1234abcd5678efgh@peer1.example.com:26656,5678efgh1234abcd@peer2.example.com:26656"

- seeds  
  Used for initial peer discovery. Connections may be dropped after peers are found.

- persistent_peers  
  Static peers that the node will continuously attempt to maintain connections with.

Restart the node after updating the configuration.

---

## TokenFactory

The `x/tokenfactory` module allows any account to create custom token denominations (e.g., `factory/lumi1.../MYTOKEN`).

### Trust Model

TokenFactory denominations are **admin-controlled assets**. The denom administrator has the following privileges:

- **Mint** — Create new tokens without limit
- **Burn** — Destroy tokens from any account
- **Force Transfer** — Move tokens between accounts without holder consent
- **Change Metadata** — Modify the denom name, symbol, and display information
- **Transfer Admin** — Reassign admin rights to another address

There is currently no admin-renounce function. Once created, a factory denom remains under admin control unless admin rights are transferred to another address.

### Important Notice for Users and Integrators

- Factory tokens (`factory/` prefix) should **not** be treated as trust-minimized or equivalent to the native chain asset (`ulwp`).
- Holders of factory tokens are subject to the trust assumptions of the denom administrator.
- Wallets, explorers, and exchanges integrating factory tokens should clearly distinguish them from native assets and inform users of the admin-controlled nature.

---

## Cosmovisor Deployment

Cosmovisor-based deployment follows the `all_new_deploy_for_cosmovisor.sh` script.

Recommended directory structure:

    ~/.lumiwave-protocold/
    └── cosmovisor
        ├── genesis/bin/lumiwave-protocold
        └── upgrades/<upgrade-name>/bin/lumiwave-protocold

---

## Releases

Testnet binaries are distributed via GitHub Releases.

Manual installation from source is recommended:

    make install

---

## Docs

- Documentation is maintained in the internal `docs/` directory.
- Can be extended to external documentation platforms such as GitBook.

---

## Contributing

- Pull requests are welcome.
- For major changes, please open an issue for discussion first.
- Commit messages should clearly describe the scope of changes.

---

## Learn more

- [Ignite CLI](https://ignite.com/cli)
- [Tutorials](https://docs.ignite.com/guide)
- [Ignite CLI docs](https://docs.ignite.com)
- [Cosmos SDK docs](https://docs.cosmos.network)
- [Developer Chat](https://discord.com/invite/ignitecli)

---

## License

Apache License 2.0
