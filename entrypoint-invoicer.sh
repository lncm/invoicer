
echo "connecting to $(LNDHOST)"
/bin/invoicer-linux-arm -lnd-host=$LNDHOST -lnd-invoice=/lnd/data/chain/bitcoin/mainnet/invoice.macaroon -lnd-readonly=/lnd/data/chain/bitcoin/mainnet/readonly.macaroon -mainnet -lnd-tls=/lnd/tls.cert
