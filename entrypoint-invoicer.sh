#!/usr/bin/env bash

/bin/invoicer -lnd-host=$LNDHOST \
    -lnd-invoice=/lnd/data/chain/bitcoin/mainnet/invoice.macaroon \
    -lnd-readonly=/lnd/data/chain/bitcoin/mainnet/readonly.macaroon \
    -lnd-tls=/lnd/tls.cert \
    -static-dir=/static/

