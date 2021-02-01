#!/bin/sh

case $NETWORK in
    simnet)
        RPCPORT=28886
        ;;
    testnet)
        RPCPORT=18886
        ;;
    mainnet)
        RPCPORT=8886
        ;;
    *)
        echo "Invalid NETWORK"
        exit 1
esac

while ! [ -e /root/.opendexd/tls.cert ]; do
    echo "Waiting for /root/.opendexd/tls.cert"
    sleep 1
done

exec bin/server --opendexd.rpchost=ond --opendexd.rpcport=$RPCPORT --opendexd.rpccert=/root/.opendexd/tls.cert \
--pairs.weight btc_usdt:4,eth_btc:3,ltc_btc:2,ltc_usdt:1
