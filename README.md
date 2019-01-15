invoicer
========

> _"fiat is irrelephant 🐘"_

### Exposes simple authenticated API on top of LND

Installing
----------

#### Easy 

Download binary from [releases page].

[releases page]: https://github.com/lncm/invoicer/releases

#### Manual

Have Go `v1.11` installed, and:

```bash
git clone https://github.com/lncm/invoicer.git
cd invoicer
make run
``` 

Usage
---

```
$ ./invoicer --help
Usage of bin/invoicer:
  -bitcoind-host string
    	Specify hostname where your bitcoind is available (default "localhost")
  -bitcoind-pass string
    	RPC password for bitcoind
  -bitcoind-port int
    	Port on which bitcoind is listening (default 8332)
  -bitcoind-user string
    	RPC user for bitcoind (default "invoicer")
  -index-file string
    	pass path to a default index file (default "static/index.html")
  -ln-client string
    	specify which LN implementation should be used. Allowed: lnd and clightning (default "lnd")
  -lnd-host string
    	Specify hostname where your lnd is available (default "localhost")
  -lnd-invoice string
    	Specify path to invoice.macaroon file (default "invoice.macaroon")
  -lnd-port int
    	Port on which lnd is listening (default 10009)
  -lnd-readonly string
    	Specify path to readonly.macaroon file (default "readonly.macaroon")
  -lnd-tls string
    	Specify path to tls.cert file (default "tls.cert")
  -port int
    	specify port to serve the website & API at (default 8080)
  -static-dir string
    	pass path to a dir containing static files to be served
  -users-file string
    	path to a file with acceptable user passwords
```

* Provide all credentials needed by LND and bitcoind,
* Make sure the certificate provided via `-lnd-tls` has used domain/IP added,
* To have `GET /history` endpoint available, make sure to pass `-users-file` path to a file containing space delimited username password pairs. Each line should contain one pair,
* By default all API paths start with `localhost:8080/api/`,
* By default all other paths serve content from path passed as `-static-dir`, 
* To keep binary running use `screen`, `tmux` or service manager of your choice

API
---

## `GET /api/info`

Returns an array of LN connstrings, ex:

```json
[
  "03935a378993d0b55056801b11957aaecb9f85f34b64245f864c22a2d25001de74@203.150.177.168:9739"
]
```

## `POST /api/payment`

Takes JSON body with two optional values:

```json
{
  "amount": 1000, 
  "desc": "payment description, also set as LN invoice description"
}
```

> **NOTE:** amount is in satoshis.

Returns payment json in a form of:

```json
{
  "created_at": 1547548139,
  "expiry": 180,
  "bolt11": "lnbc10m1pwrmd0tpp5zqkst04uexshsyf2km4e9hyuc9r9rkvn6w94else0x2ffj0u98jqdq2v3sk66tpdccqzysxqz958kapl4hfq5uq6nelt93c6wvferkyj29v89sr2mlm7x9kecq02s2phgq30fq77wukzasnksngty0qd6lz4tsaz0h7tfyqj9pcp06wd9cql8gp5w",
  "hash": "102d05bebcc9a178112ab6eb92dc9cc14651d993d38b5cfe19799494c9fc29e4",
  "address": "3MgiKgMY1ZxRNrLyhYPJpNLPb37TxkrJrb"
}
```

> **NOTE:** `created_at` is a unix timestamp. `expiry` is in seconds.

On error, returns:

```json
{
  "error": "content of an error"
}
```

## `GET /api/payment?hash=LN-hash&address=BTC-address`

Takes two parameters:

* `hash` - hash of the preimage returned previously by the `POST /payment` endpoint
* `address` - Bitcoin address returned previously by the `POST /payment` endpoint

> **NOTE:** providing just one will run checks on one network only.

Returns:

!<br>
!<br>
TBD !<br>
!<br>
!

## `GET /api/history`

Presently takes no arguments, and returns:


!<br>
!<br>
TBD !<br>
!<br>
!


## `GET /api/healthcheck`

Checks whether connections to lnd and bitcoind work.

Returns code `200` if they do.

Returns code `500`, and a text of the first found error, if they don't.  

Deployment
---

To deploy the binary to your Raspberry Pi, run (replacing all values with ones specific to your setup):

```bash
$ make deploy REMOTE_USER=root REMOTE_HOST=pi-hdd REMOTE_DIR=/home/ln/bin/ 
``` 

If you want to expose tips page (`common/index.html`), make sure to expose port `:1666` on your firewall. The page will be located at path root. 


Development
---
All contributions are welcome.

Feel free to get in touch!

---

Made with 🥩 in Chiang Mai
