#!/bin/bash

set -o errexit # -e
set -o nounset # -u
set -o pipefail
set -o monitor # -m

#XUD_DIR=$HOME/.opendex
XUD_DIR="${XUD_DIR:-$HOME/.opendex}"
XUD_CONF="${XUD_CONF:-$XUD_DIR/opendex.conf}"
TOR_DIR="${TOR_DIR:-$XUD_DIR/tor}"
TOR_DATA_DIR="${TOR_DATA_DIR:-$XUD_DIR/tor-data}"
TOR_TORRC="${TORRC:-/etc/tor/torrc}"

LND_HOSTNAME_FILE="$TOR_DIR/hostname"

case $NETWORK in
    mainnet)
        DEFAULT_P2P_PORT=8885
        DEFAULT_RPC_PORT=8886
        DEFAULT_HTTP_PORT=8887
        ;;
    testnet)
        DEFAULT_P2P_PORT=18885
        DEFAULT_RPC_PORT=18886
        DEFAULT_HTTP_PORT=18887
        ;;
    simnet)
        DEFAULT_P2P_PORT=28885
        DEFAULT_RPC_PORT=28886
        DEFAULT_HTTP_PORT=28887
        ;;
    *)
        echo >&2 "Error: Unsupported network: $NETWORK"
        exit 1
esac

P2P_PORT="${P2P_PORT:-DEFAULT_P2P_PORT}"
RPC_PORT="${RPC_PORT:-DEFAULT_RPC_PORT}"
HTTP_PORT="${HTTP_PORT:-DEFAULT_HTTP_PORT}"


[[ -e ${TOR_TORRC} ]] || cat <<EOF >/etc/tor/torrc
DataDirectory $TOR_DATA_DIR
ExitPolicy reject *:* # no exits allowed
HiddenServiceDir $TOR_DIR
HiddenServicePort $P2P_PORT 127.0.0.1:$P2P_PORT
HiddenServiceVersion 3
EOF

tor -f $TOR_TORRC &

if [[ -z "$LND_HOSTNAME_FILE" ]]
    while [[ ! -e "$LND_HOSTNAME_FILE" ]]; do
        echo "[entrypoint] Waiting for opendexd onion address at $LND_HOSTNAME_FILE"
        sleep 1
    done

    XUD_ADDRESS=$(cat "$LND_HOSTNAME_FILE")
fi

echo "[entrypoint] Onion address for opendexd is $XUD_ADDRESS"


echo '[entrypoint] Detecting localnet IP for lndbtc...'
LNDBTC_IP=$(getent hosts lndbtc || echo '' | awk '{ print $1 }')
echo "$LNDBTC_IP lndbtc" >> /etc/hosts

echo '[entrypoint] Detecting localnet IP for lndltc...'
LNDLTC_IP=$(getent hosts lndltc || echo '' | awk '{ print $1 }')
echo "$LNDLTC_IP lndltc" >> /etc/hosts

echo '[entrypoint] Detecting localnet IP for connext...'
CONNEXT_IP=$(getent hosts connext || echo '' | awk '{ print $1 }')
echo "$CONNEXT_IP connext" >> /etc/hosts


LNDBTC_TLS_CERT="${LNDBTC_TLS_CERT:-/root/.lndbtc/tls.cert}"
while [[ ! -e "$LNDBTC_TLS_CERT" ]]; do
    echo "[entrypoint] Waiting for ${LNTBTC_TLS_CERT} to be created..."
    sleep 1
done

LNDLTC_TLS_CERT="${LNDLTC_TLS_CERT:-/root/.lndltc/tls.cert}"
while [[ ! -e "$LNDLTC_TLS_CERT" ]]; do
    echo "[entrypoint] Waiting for ${LNTLTC_TLS_CERT} to be created..."
    sleep 1
done


[[ -e $XUD_CONF && $PRESERVE_CONFIG == "true" ]] || {
    cp /app/sample-opendex.conf $XUD_CONF

    sed -i "s/network.*/network = \"$NETWORK\"/" $XUD_CONF
    sed -i 's/noencrypt.*/noencrypt = false/' $XUD_CONF
    sed -i '/\[http/,/^$/s/host.*/host = "0.0.0.0"/' $XUD_CONF
    sed -i "/\[http/,/^$/s/port.*/port = $HTTP_PORT/" $XUD_CONF
    sed -i '/\[lnd\.BTC/,/^$/s/host.*/host = "lndbtc"/' $XUD_CONF
    sed -i "/\[lnd\.BTC/,/^$/s|^$|certpath = \"/root/.lndbtc/tls.cert\"\nmacaroonpath = \"/root/.lndbtc/data/chain/bitcoin/$NETWORK/admin.macaroon\"\n|" $XUD_CONF
    sed -i '/\[lnd\.LTC/,/^$/s/host.*/host = "lndltc"/' $XUD_CONF
    sed -i '/\[lnd\.LTC/,/^$/s/port.*/port = 10009/' $XUD_CONF
    sed -i "/\[lnd\.LTC/,/^$/s|^$|certpath = \"/root/.lndltc/tls.cert\"\nmacaroonpath = \"/root/.lndltc/data/chain/litecoin/$NETWORK/admin.macaroon\"\n|" $XUD_CONF
    sed -i "/\[p2p/,/^$/s/addresses.*/addresses = \[\"$XUD_ADDRESS\"]/" $XUD_CONF
    sed -i "/\[p2p/,/^$/s/port.*/port = $P2P_PORT/" $XUD_CONF
    sed -i '/\[p2p/,/^$/s/tor = .*/tor = true/' $XUD_CONF
    sed -i '/\[p2p/,/^$/s/torport.*/torport = 9050/' $XUD_CONF
    sed -i '/\[raiden/,/^$/s/disable.*/disable = true/' $XUD_CONF
    sed -i '/\[rpc/,/^$/s/host.*/host = "0.0.0.0"/' $XUD_CONF
    sed -i "/\[rpc/,/^$/s/port.*/port = $RPC_PORT/" $XUD_CONF
    sed -i '/\[connext/,/^$/s/disable.*/disable = false/' $XUD_CONF
    sed -i '/\[connext/,/^$/s/host.*/host = "connext"/' $XUD_CONF
    sed -i '/\[connext/,/^$/s/port.*/port = 8000/' $XUD_CONF
    sed -i '/\[connext/,/^$/s/webhookhost.*/webhookhost = "opendexd"/' $XUD_CONF
    sed -i "/\[connext/,/^$/s/webhookport.*/webhookport = $HTTP_PORT/" $XUD_CONF
}

echo "[entrypoint] Launch with opendexd.conf:"
cat $XUD_CONF

XUD_BACKUP_DIR="${XUD_BACKUP_DIR:-/root/backup}" /opendexd-backup.sh &

# use exec to properly respond to SIGINT
exec opendexd $@
