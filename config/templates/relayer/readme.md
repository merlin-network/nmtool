# relayer

this directory contains the configuration for the [relayer](https://github.com/Fury-Labs/relayer)
which is the service responsible for the initial setup of the ibc channel between `nemo` and the
`ibcchain` spun up by the `--ibc` flag of the `testnet bootstrap` command.

## setting up the configuration

if you ever need to initialize this configuration again, these are the steps.
here we use `v2.2.0`, published in a container to docker hub. to build & deploy updated versions,
see "build & deploy new relayer images".

1. start up a local nemo & ibc chain
  * generate config for nemo and ibcchain: `nmtool testnet gen-config nemo --ibc`
  * run the networks: `nmtool testnet up -d`
2. create a temporary directory to build the configuration in. here, we use `$PWD/temp`
  * `mkdir temp`
3. we use the docker container to run `rly`, the relayer's cli. initialize the config by running the following commands:
```bash
# initialize the config
docker run -v $PWD/temp:/home/relayer/.relayer --network generated_default nemo/relayer:v2.2.0 rly config init
# add nemo chain. this will init the nemo config to the defaults defined in https://github.com/cosmos/chain-registry
docker run -v $PWD/temp:/home/relayer/.relayer --network generated_default nemo/relayer:v2.2.0 rly chains add nemo
```
open `temp/config.yaml`. copy & duplicate the `chains.nemo` object and name it `ibcchain`.
Then make the following updates:
* update both chain's `key` values to `testkey`. we will create these keys next.
* update `ibcchain` values:
  * `chain-id`: highbury_710-2
  * `rpc-addr`: http://ibcnode:26657
  * `gas-prices`: 0.01uatom (the primary denom of `ibcchain` is `uatom`)
* update `nemo` values:
  * `chain-id`: highbury_710-1
  * `rpc-addr`: http://furynode:26657

here's the diff:
```diff
10,12c10,12
<       key: default
<       chain-id: fury_4200-10
<       rpc-addr: <some-https-rpc-mainnet-node-address>
---
>       key: testkey
>       chain-id: highbury_710-2
>       rpc-addr: http://ibcnode:26657
16c16
<       gas-prices: 0.01ufury
---
>       gas-prices: 0.01uatom
26,28c26,28
<       key: default
<       chain-id: fury_4200-10
<       rpc-addr: <some-https-rpc-mainnet-node-address>
---
>       key: testkey
>       chain-id: highbury_710-1
>       rpc-addr: http://furynode:26657
```
4. generate the keys used by the relayer. we use the `relayer` account from [`addresses.json`](../../common/addresses.json):
```bash
export RELAYER_MFURYNIC='very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food'
# add the key for nemo
docker run -v $PWD/temp:/home/relayer/.relayer --network generated_default nemo/relayer:v2.2.0 rly keys restore --coin-type 459 nemo testkey "$RELAYER_MFURYNIC"
# add the key for ibcchain
docker run -v $PWD/temp:/home/relayer/.relayer --network generated_default nemo/relayer:v2.2.0 rly keys restore --coin-type 459 ibcchain testkey "$RELAYER_MFURYNIC"
```

That's it! Verify your configured relayer accounts have a balance on each chain:
```
$ docker run -v $PWD/temp:/home/relayer/.relayer --network generated_default nemo/relayer:v2.2.0 rly q balance nemo
address {fury1ypjp0m04pyp73hwgtc0dgkx0e9rrydech3f8g4} balance {1000000000ufury}

$ docker run -v $PWD/temp:/home/relayer/.relayer --network generated_default nemo/relayer:v2.2.0 rly q balance ibcchain
address {fury1ypjp0m04pyp73hwgtc0dgkx0e9rrydech3f8g4} balance {1000000000uatom}
```

5. Copy the directories to the template
```bash
rm -fr config/templates/relayer/config config/templates/relayer/keys
mv temp/config config/templates/relayer/config
mv temp/keys config/templates/relayer/keys
```
Note that there will always be a diff when regenerating the keys, as the create time is baked into the JWT.

## build & deploy relayer images

1. Checkout the [relayer repo](https://github.com/Fury-Labs/relayer).
Fetch & merge [upstream](https://github.com/cosmos/relayer) changes if necessary.
2. Checkout the desired tag: `git checkout $TAG`
3. cross-platform build & push the images as the correct tag:
`docker buildx build --platform=linux/amd64,linux/arm64 . -t nemo/relayer:$TAG --push`

Note that you will need docker hub permission for the nemo org. Additionally, you will need buildx
configured. Initial setup of docker buildx: `docker buildx create --use`
