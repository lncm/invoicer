#!/usr/bin/env bash

create_default_users_list() {
    /bin/echo "invoicer defaultpassw0rd" > /users.list
    USERSFILE='/users.list'
}

# Default values
if [ -z $LNCLIENT ]; then
    LNCLIENT='lnd'
fi
if [ -z $BTCHOST ]; then
    BTCHOST='btcbox'
fi
if [ -z $BTCRPCUSER ]; then
    BTCRPCUSER='invoicer'
fi
if [ -z $BTCRPCPASS ]; then
    BTCRPCPASS='password'
fi
if [ -z $BTCRPCPORT ]; then
    BTCRPCPORT=8332
fi

if [ -z $LNDHOST ]; then
    LNDHOST='lightningbox'
fi
if [ -z $LNDPORT ]; then
    LNDPORT=10009
fi
if [ -z $PORT ]; then
    PORT=8080
fi
if [ -z $READONLYMACAROON ]; then
    READONLYMACAROON='/lnd/data/chain/bitcoin/mainnet/readonly.macaroon'
fi
if [ -z $INVOICEMACAROON ]; then
    INVOICEMACAROON='/lnd/data/chain/bitcoin/mainnet/invoice.macaroon'
fi
if [ -z $LNDTLSFILE ]; then
    LNDTLSFILE='/lnd/tls.cert'
fi
if [ -z $STATICDIR ]; then
    STATICDIR='/static/'
fi
if [ -z $USERSFILE ]; then
    create_default_users_list
else
    # If its set, check if the file exists otherwise just set it and make a new one
    if ! [ -f $USERSFILE ]; then
        create_default_users_list
    fi
fi

/bin/invoicer -ln-client=$LNCLIENT \
    -lnd-host=$LNDHOST \
    -lnd-port=$LNDPORT \
    -port=$PORT \
    -lnd-invoice=$INVOICEMACAROON \
    -lnd-readonly=$READONLYMACAROON \
    -lnd-tls=$LNDTLSFILE \
    -bitcoind-host=$BTCHOST \
    -bitcoind-user=$BTCRPCUSER \
    -bitcoind-pass=$BTCRPCPASS \
    -bitcoind-port=$BTCRPCPORT \
    -users-file=$USERSFILE \
    -static-dir=$STATICDIR
