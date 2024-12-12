#!/bin/bash -e

# cleanup old data
rm -rf $HOME/.tacchaind

# environment variables
TACCHAIND=${TACCHAIND:-$(which tacchaind)}
DENOM=${DENOM:-utac}
CHAIN_ID=${CHAIN_ID:-tacchain_2390-1}
KEYRING_BACKEND=${KEYRING_BACKEND:-test}

# set cli options default values
$TACCHAIND config set client chain-id $CHAIN_ID
$TACCHAIND config set client keyring-backend $KEYRING_BACKEND
$TACCHAIND config set client output json

# init genesis file
$TACCHAIND init test --chain-id $CHAIN_ID --default-denom $DENOM

# set evm_denom to $DENOM in genesis
sed -i.bak "s/\"evm_denom\": \"aphoton\"/\"evm_denom\": \"$DENOM\"/g" $HOME/.tacchaind/config/genesis.json
sed -i.bak "s/\"no_base_fee\": false/\"no_base_fee\": true/g" $HOME/.tacchaind/config/genesis.json
# set max gas which is required for evm txs
sed -i.bak "s/\"max_gas\": \"-1\"/\"max_gas\": \"20000000\"/g" $HOME/.tacchaind/config/genesis.json
# enable evm eip-3855
sed -i.bak "s/\"extra_eips\": \[\]/\"extra_eips\": \[\"3855\"\]/g" $HOME/.tacchaind/config/genesis.json

# setup and add validator to genesis
$TACCHAIND keys add validator
$TACCHAIND genesis add-genesis-account validator 100000000000000000000000000$DENOM --keyring-backend $KEYRING_BACKEND
$TACCHAIND genesis gentx validator 1000000$DENOM --chain-id $CHAIN_ID
$TACCHAIND genesis collect-gentxs