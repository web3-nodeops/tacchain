#!/bin/bash

CHAIN_ID=${CHAIN_ID:-tacchain_2390-1}
TACCHAIND=${TACCHAIND:-$(which tacchaind)}

$TACCHAIND start --chain-id $CHAIN_ID
