#!/bin/sh
# This script launches ETH2 beacon clients for Rocket Pool's docker stack; only edit if you know what you're doing ;)


# Lighthouse startup
if [ "$CLIENT" = "lighthouse" ]; then

    exec /usr/local/bin/lighthouse beacon --network pyrmont --datadir /ethclient/lighthouse --port 9091 --discovery-port 9091 --eth1 --eth1-endpoint "$ETH1_PROVIDER" --http --http-address 0.0.0.0 --http-port 5052 --target-peers 25 --metrics --metrics-address 0.0.0.0 --metrics-port 5053 --http-allow-origin '*'

fi


# Prysm startup
if [ "$CLIENT" = "prysm" ]; then

    exec /app/beacon-chain/beacon-chain --accept-terms-of-use --pyrmont --datadir /ethclient/prysm --p2p-tcp-port 9091 --p2p-udp-port 9091 --http-web3provider "$ETH1_PROVIDER" --rpc-host 0.0.0.0 --rpc-port 5052 --monitoring-host 0.0.0.0 --blst

fi

