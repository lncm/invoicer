#!/usr/bin/env bash

if [ -z $INVOICER_HOSTPORT ]; then
    INVOICER_HOSTPORT='localhost:1666'
fi

CHECK_INVOICER=`curl "http://$INVOICER_HOSTPORT/api/healthcheck" -D headerfile 2>/dev/null; head -1 headerfile | grep -c 200 ; rm headerfile`
if [ $CHECK_INVOICER == 1 ]; then
    echo "Online"
    exit 0
 else
    echo "Not Online"
    exit 1
fi
