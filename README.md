# nmtool

Assorted dev tools for working with the nemo blockchain.

To get started with running a local nemo network, check out our docs on [Getting Started](https://docs.nemo.io/docs/cosmos/getting-started).

## Installation

```bash
make install
```

## Initialization: nmtool testnet

Note that the most accurate documentation lives in the CLI itself. It's recommended you read through `nmtool testnet bootstrap --help`.

Option 1:

The `nmtool testnet bootstrap` command starts a local Fury blockchain as a
background docker container called `generated-furynode-1`. The bootstrap command
only starts the Fury blockchain and Fury REST server services.

```bash
# Start new testnet
nmtool testnet bootstrap --nemo.configTemplate master
```

The endpoints are exposed to localhost:

* RPC: http://localhost:26657
* REST: http://localhost:1317
* GRPC: http://localhost:9090
* GRPC Websocket: http://localhost:9091
* EVM JSON-RPC: http://localhost:8545
* EVM Websocket: http://localhost:8546

Option 2:

To generate a testnet for nemo, binance chain, and a deputy that relays swaps between them:

```bash
# Generate a new nmtool configuration based off template files
nmtool testnet gen-config nemo binance deputy --nemo.configTemplate master

# Pull latest docker images. Docker must be running.
cd ./full_configs/generated && docker-compose pull

# start the testnet
nmtool testnet up

# When finished with usage, shut down the processes
nmtool testnet down
```

### Flags

Additional flags can be added when initializing a testnet to add additional
services:

`--ibc`: Run Fury testnet with an additional IBC chain. The IBC chain runs in the container named `ibcnode`. It has primary denom `ufury`.

Example:

```bash
# Run Fury testnet with an additional IBC chain
nmtool testnet bootstrap --nemo.configTemplate master --ibc
```

`--geth`: Run a go-ethereum node alongside the Fury testnet. The geth node is
initialized with the Fury Bridge contract and test ERC20 tokens. The Fury EVM
also includes Multicall contracts deployed. The contract addresses can be found
on the [Fury-Labs/nemo-bridge](https://github.com/Fury-Labs/nemo-bridge#development)
README.

Example:

```bash
# Run the testnet with a geth node in parallel
nmtool testnet bootstrap --nemo.configTemplate master --geth
```

Geth node ports are **not** default, as the Fury EVM will use default JSON-RPC
ports:

Fury EVM RPC Ports:

* HTTP JSON-RPC: `8545`
* WS-RPC port: `8546`

Geth RPC Ports:

* HTTP JSON-RPC: `8555`
* WS-RPC port: `8556`

To connect to the associated Ethereum wallet with Metamask, setup a new network with the following parameters:
* New RPC URL: `http://localhost:8555`
* Chain ID: `88881` (configured from the [genesis](config/templates/geth/initstate/genesis.json#L3))
* Currency Symbol: `ETH`

Finally, connect the mining account by importing the JSON config in [this directory](config/templates/geth/initstate/.geth/keystore)
with [this password](config/templates/geth/initstate/eth-password).

## Automated Chain Upgrade

Futool supports running upgrades on a chain. To do this requires the nemo final docker image to have a registered upgrade handler.
The upgrade will start a chain with the docker container tag from `--upgrade-base-image-tag`. Once it reaches height `--upgrade-height`, it halts the chain for an upgrade named `--upgrade-name`. At that point, the container is restated with the desired container: `FURY_TAG` if defined, of if not defined, the default tag for the config template.

**Example**:
Test a chain upgrade from v0.19.2 -> v0.21.0 at height 15.

Using an overridden docker image tag:
```
$ FURY_TAG=v0.21.0 nmtool testnet bootstrap --upgrade-name v0.21.0 --upgrade-height 15 --upgrade-base-image-tag v0.19.2
```

Using a config template:
```
Test a chain upgrade from v0.19.2 -> v0.21.0:
$ nmtool testnet bootstrap --nemo.configTemplate v0.21 --upgrade-name v0.21.0 --upgrade-height 15 --upgrade-base-image-tag v0.19.2
```

## Usage: nmtool testnet

REST APIs for both blockchains are exposed on localhost:

- Fury: http://localhost:1317
- Binance Chain: http://localhost:8080

You can also interact with the blockchain using the `nemo` command line. In a
new terminal window, set up an alias to `nemo` on the dockerized nemo node and
use it to send a query.

```bash
# Add an alias to the dockerized nemo cli
alias dfury='docker exec -it generated_furynode_1 nemo'

# Confirm that the alias has been added
alias nemo

# For versions before v0.16.x
alias dkvcli='docker exec -it generated_furynode_1 kvcli'
```

Note that for some architectures or docker versions, the containers are generated with hyphens (`-`) instead of underscores (`_`).

You can test the set up and alias by executing a sample query:

```bash
dfury status
dfury q cdp params
```

The chain has several accounts that are funded from genesis. A list of the account names can be found [here](config/common/addresses.json).

The binary is pre-configured to have these keys in its keyring so you should be able to use them directly.
```bash
# Example sending funds from `whale` to another account
dfury tx bank send whale [nemo-address-to-fund] 1000000ufury --gas-prices 0.001ufury -y

# Check transaction result by tx hash
dfury q tx [tx-hash]
```
### A note about eth accounts

Account keys can be created with two different algorithms in Fury: `secp256k1` and `eth_secp256k1`.
Which algorithm is used is dictate by the presence of the `--eth` flag on key creation.

Eth accounts can be exported for use in ethereum wallets like Metamask. A list of of the pre-funded eth accounts can be found [here](config/generate/genesis/auth.accounts/eth-accounts.json).
Notable, `whale2` is an eth account. These keys can be easily imported into a wallet via their private keys:
```bash
# DANGEROUS EXPORT OF PRIVATE KEY BELOW! BE CAREFUL WITH YOUR PRIVATE KEYS FOR MAINNET ACCOUNTS.
dfury keys unsafe-export-eth-key whale2
```
The above will output the hex-encoded ethereum private key that can be directly imported to Metamask or another EVM-supporting wallet.

You can always import or generate new eth accounts as well:
```bash
# generate new account
dfury keys add new-eth-account --eth

# recover an eth account from a mnemonic
dfury keys add new-eth-account2 --eth --recover
eth flag specified: using coin-type 60 and signing algorithm eth_secp256k1
> Enter your bip39 mnemonic
# enter your mnemonic here

# import an eth account from a hex-encoded ethereum private key
nemo keys unsafe-import-eth-key new-eth-account3 [priv-key]
```

### ERC20 token

The master template includes a pre-deployed ERC20 token with the name "USD Coin". The token is configured to be converted to an sdk coin of the denom `erc20/multichain/usdc`.

Token Address: `0xeA7100edA2f805356291B0E55DaD448599a72C6d`
Funded Account: `whale2` - `0x03db6b11F47d074a532b9eb8a98aB7AdA5845087` (1000 USDC)

## Shut down: nmtool testnet

When you're done make sure to shut down the nmtool testnet. Always shut down the nmtool testnets before pulling the latest image from docker, otherwise you may experience errors.

```bash
nmtool testnet down
```

# Updating nemo genesis

When new versions of nemo are released, they often involve changes to genesis.
The nemo `master` template includes a genesis.json that is generated from a pure state:
* Ensure the desired version of `nemo` is in your path as `nemo`
* Run `make generate-nemo-genesis`
* The script will create a genesis with desired accounts & validator
* Updates to the genesis should be made in [`update-nemo-genesis.sh`](./config/generate/genesis/generate-nemo-genesis.sh)
