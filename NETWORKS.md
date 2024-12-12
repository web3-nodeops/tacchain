# Tac Networks

| Chain ID        | Type      | Status     | Version  | Notes         |
|-----------------|-----------|------------|----------|---------------|
| tacchain_2390-1 | `testnet` | **Active** | `v0.0.1` | Turin Testnet |

## Turin Testnet (`tacchain_2390-1`)

| Chain ID                    | `tacchain_2390-1`                                                                             |
|-----------------------------|-----------------------------------------------------------------------------------------------|
| Tacchaind version           | `v0.0.1`                                                                                      |
| RPC                         | <https://newyork-inap-72-251-230-233.ankr.com/tac_tacd_testnet_full_tendermint_rpc_1>         |
| Genesis                     | <https://newyork-inap-72-251-230-233.ankr.com/tac_tacd_testnet_full_tendermint_rpc_1/genesis> |
| gRPC                        | <https://newyork-inap-72-251-230-233.ankr.com/tac_tacd_testnet_full_tendermint_grpc_web_1>    |
| REST API                    | <https://newyork-inap-72-251-230-233.ankr.com/tac_tacd_testnet_full_tendermint_rest_1>        |
| EVM JSON RPC                | <https://newyork-inap-72-251-230-233.ankr.com:443/tac_tacd_testnet_full_rpc_1>                |
| Faucet                      | <https://faucet.tac-turin.ankr.com>                                                           |
| EVM Explorer                | <https://explorer.tac-turin.ankr.com>                                                         |
| Cosmos Explorer             | <https://explorer.tacchain-turin.ankr.com/tac>                                                |
| Timeout commit (block time) | 3s                                                                                            |
| Peer 1                      | 9b4995a048f930776ee5b799f201e9b00727ffcc@107.6.94.246:45120                                   |
| Peer 2                      | e3c2479a6f418841bd64bae6dff027ea3efc1987@72.251.230.233:45120                                 |
| Peer 3                      | fbf04b3d67705ed48831aa80ebe733775e672d1a@107.6.94.246:45110                                   |
| Peer 4                      | 5a6f0e342ea66cb769194c81141ffbff7417fbcd@72.251.230.233:45110                                 |
| Snapshots                   | TBD                                                                                           |
| Frontend                    | TBD                                                                                           |

## Join a network

This example guide connects to testnet. You can replace `chain-id`, `persistent_peers`, `timeout_commit`, `genesis url` with the network you want to join. `--home` flag specifies the path to be used. The example will create [.testnet](.testnet) folder.

### Prerequisites

  - [Go >= v1.21](https://go.dev/doc/install)
  - jq
  - curl

### 1. Install `tacchaind`

``` sh
make install
```

### 2. Initialize network folder

``` sh
tacchaind init testnode --chain-id tacchain_2390-1 --home .testnet
```

### 3. Modify your [config.toml](.testnet/config/config.toml)

``` toml
..
timeout_commit = "3s"
..
persistent_peers = "9b4995a048f930776ee5b799f201e9b00727ffcc@107.6.94.246:45120,e3c2479a6f418841bd64bae6dff027ea3efc1987@72.251.230.233:45120,fbf04b3d67705ed48831aa80ebe733775e672d1a@107.6.94.246:45110,5a6f0e342ea66cb769194c81141ffbff7417fbcd@72.251.230.233:45110"
..
```

### 4. Fetch genesis

``` sh
curl https://newyork-inap-72-251-230-233.ankr.com/tac_tacd_testnet_full_tendermint_rpc_1/genesis | jq '.result.genesis' > .testnet/config/genesis.json
```

### 5. Start node

``` sh
tacchaind start --chain-id tacchain_2390-1 --home .testnet
```

## Join as a validator

### 1. Make sure you followed [Join a network](#join-a-network) guide and you have a fully synced node to the latest block.

### 2. Fund account and import key

1. Use the [faucet](https://faucet.tac-turin.ankr.com/) to get funds.

2. Export your metamask private key

3. Import private key using the following command. Make sure to replace `<PRIVATE_KEY>` with your funded private key.

``` sh
tacchaind --home .testnet keys unsafe-import-eth-key validator <PRIVATE_KEY> --keyring-backend test
```

### 3. Send `MsgCreateValidator` transaction

1. Generate tx json file

``` sh
echo "{\"pubkey\":$(tacchaind --home .testnet tendermint show-validator),\"amount\":\"1000000000utac\",\"moniker\":\"testnode\",\"identity\":null,\"website\":null,\"security\":null,\"details\":null,\"commission-rate\":\"0.1\",\"commission-max-rate\":\"0.2\",\"commission-max-change-rate\":\"0.01\",\"min-self-delegation\":\"1\"}" > validatortx.json
```

2. Broadcast tx

``` sh
tacchaind --home .testnet tx staking create-validator validatortx.json --from validator --keyring-backend test -y
```

### 4. Delegate more tokens (optional)

``` sh
tacchaind --home .testnet tx staking delegate $(tacchaind --home .testnet q staking validators --output json | jq -r '.validators[] | select(.description.moniker == "testnode") | .operator_address') 1000000000utac --keyring-backend test --from validator -y
```

## FAQ

**1) I need some funds on the `tacchain_2390-1` testnet, how can I get them?**

You can request testnet tokens for the `tacchain_2390-1` testnet from the faucet available at <https://faucet.tac-turin.ankr.com/>. Please note that the faucet currently dispenses up to 10 TAC per day per address.

**2) I have completed the guide to join as a validator, but my node is not in the active validator set?**

In order to be included in the active validator set, your validator must have atleast 1 voting power, or if the maximum validators limit has been reached your validator must have greater amount of TAC delegated to them than the validator with lowest amount delegated. Read more - https://forum.cosmos.network/t/why-is-my-newly-created-validator-unbonded/1841/2
