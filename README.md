lncm/invoicer
==============

![Build Status]
[![gh_last_release_svg]][gh_last_release_url]
[![Docker Image Size]][invoicer-docker-hub]
[![Docker Pulls Count]][invoicer-docker-hub]
[![Maintainability]][codeclimate-maintainability]

[Build Status]: https://github.com/lncm/invoicer/workflows/Build%20%26%20deploy%20invoicer%20on%20a%20git%20tag%20push/badge.svg

[gh_last_release_svg]: https://img.shields.io/github/v/release/lncm/invoicer?sort=semver
[gh_last_release_url]: https://github.com/lncm/invoicer/releases/latest

[Docker Image Size]: https://img.shields.io/microbadger/image-size/lncm/invoicer.svg
[Docker Pulls Count]: https://img.shields.io/docker/pulls/lncm/invoicer.svg?style=flat
[invoicer-docker-hub]: https://hub.docker.com/r/lncm/invoicer

[Maintainability]: https://api.codeclimate.com/v1/badges/02fbc85043d086e318a4/maintainability
[codeclimate-maintainability]: https://codeclimate.com/github/lncm/invoicer/maintainability


> _"fiat is irrelephant 🐘"_

### Exposes simple API to receive payments on top of LND

Install
---

#### Easy 

Download binary from [Github Releases].

Or

Pull Docker image from [Docker Hub].

```shell script
docker pull lncm/invoicer
```

[Github Releases]: https://github.com/lncm/invoicer/releases
[Docker Hub]: https://hub.docker.com/r/lncm/invoicer


#### Manual

Have Go `v1.13` installed, and:

```bash
git clone https://github.com/lncm/invoicer.git
cd invoicer
make run
``` 

**NOTE:** It **might** work on previous versions of Go, but only the latest is supported.


Usage
---

```
$ ./invoicer --help
Usage of bin/invoicer:
  -config string
    	Path to a config file in TOML format (default "~/.lncm/invoicer.conf")
```

**NOTE:** Before running make sure `invoicer.conf` exists somewhere.  To see what's expected in it, please refer to `invoicer.example.conf` file.

* Provide all credentials needed by LND and bitcoind,
* Or (if you use ex. neutrino) disable bitcoind dependency by adding: `off-chain-only=true` to config,
* Make sure the certificate provided via `tls = ` in `[lnd]` section has your domain/IP added,
* To have `GET /history` endpoint available, make sure to add `user = "password"` pairs to `[users]` section,
* By default, all API paths start with `localhost:8080/api/`,
* By default, all other paths serve content from path passed as `static-dir = `,
* To keep binary running use `screen`, `tmux`, `Docker` or service manager of your choice,
* Logging to file can be disabled by putting `log-file=none` to `invoicer.conf`.

Docker
---

Create `invoicer.conf` file in the directory you're in. See `invoicer.example.conf` for inspiration.  Also copy `tls.cert`, `invoice.macaroon`, and `readonly.macaroon` from lnd directory to the dir you're in, and:

Run:

```bash
# Replace /path/to/lnd with your LND directory (where the macaroons live)
docker run -it --rm \
    -v $(pwd)/:/data/ \
    -v /path/to/lnd:/lnd/ \
    -p 8080:8080 \
    --name invoicer \
    --detach \
    lncm/invoicer:v0.8.1
```


API
---

## `GET /`

If `static-dir =` passed, serves `index.html` located within there. Otherwise 404. 

## `GET /api/info`

Returns node info, ex:

```json
{
  "on-chain": true,
  "off-chain": true,
  "uris": [
    "03935a378993d0b55056801b11957aaecb9f85f34b64245f864c22a2d25001de74@202.44.225.68:9739"
  ]
}
```


## `POST /api/payment`

Takes JSON body with three optional values:

```json
{
  "amount": 1000, 
  "desc": "payment description, also set as LN invoice description",
  "only": "btc|ln"
}
```

> **NOTE:** `amount` is in satoshis.

> **NOTE_2:** `only` if specified, can only be `btc` or `ln`.  

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

#### Takes:

* `hash` (string) - hash of the preimage returned previously by the `POST /payment` endpoint
* `address` (string) - Bitcoin address returned previously by the `POST /payment` endpoint
* `flexible` (bool) - If set, will ignore amount checks.  Useful for accepting donations, where there's no "too much" or "too little". 

> **NOTE:** providing just one of `address` or `hash` will run checks on one network only.

#### Returns:

##### on expiry (code 408)

```json
{
  "error": "expired"
}
```

##### on LN success (code 200)

```json
{
    "ln": {
        "created_at": 1547562917,
        "is_paid": true,
        "expiry": 180,
        "amount": 1000
    }
}
```

##### on BTC success w/exact amount (code 200)

```json
{
    "bitcoin": {
        "address": "3Ee7SdoCCC3ECC3NAPx5VwE6F8pjwnZzpW",
        "amount": 0.0001,
        "confirmations": 0,
        "txids": [
            "9faf2560c1a43599abaad06ab4d038ff7353c4f2992fe44ccba20fd25d6d3a60"
        ]
    }
}
```

##### on BTC success w/too big amount (code 202)

> **NOTE:** This can only happen if `flexible` is not set.

```json
{
    "bitcoin": {
        "address": "3Ee7SdoCCC3ECC3NAPx5VwE6F8pjwnZzpW",
        "amount": 0.0001,
        "confirmations": 0,
        "txids": [
            "9faf2560c1a43599abaad06ab4d038ff7353c4f2992fe44ccba20fd25d6d3a60"
        ]
    }
}
``` 

##### on BTC success w/too small amount (code 402)

> **NOTE:** This can only happen if `flexible` is not set.

```json
{
    "error": "not enough",
    "bitcoin": {
        "address": "3Ee7SdoCCC3ECC3NAPx5VwE6F8pjwnZzpW",
        "amount": 0.0001,
        "confirmations": 0,
        "txids": [
            "9faf2560c1a43599abaad06ab4d038ff7353c4f2992fe44ccba20fd25d6d3a60"
        ]
    }
}
```

##### On any other error
```json
{
    "error": "error message…"
}
```


## `GET /api/history`

#### Takes:

* `only_status` - filter payments to only one specific state: `paid`, `expired` or `panding`.

#### Returns <small>(various cases included below):</small>

```json
{
  "history": [
    {
      "created_at": 1547549353,
      "expiry": 180,
      "bolt11": "lnbc1u1pwrmw4fpp5hvyhndhrc0uxzvzgzpcyzsr6z0mkm8w6fwqmkhcluy6typw7ukdqdqcw3jhxapqf38zqurp09kk2mn5cqzysxqz95adjqqm2v7dhkhaehng6rrht9xafyqxuea6fus3rluzmjutjnskdzckasq958hku9t5lmvag6g2jn3fczx8gpep6qqm5aft4p07wwy6sq28p740",
      "hash": "bb0979b6e3c3f8613048107041407a13f76d9dda4b81bb5f1fe134b205dee59a",
      "address": "3K4PFPchtzzPXL5aHheJLi79qjh6hn3csv",
      "description": "test LN payment",
      "amount": 100,
      "is_expired": true,
      "is_paid": true,
      "ln_paid": true,
      "btc_paid": false,
      "btc_amount": 0,
      "confirmations": 0,
      "txids": []
    },
    {
      "created_at": 1547549409,
      "expiry": 180,
      "bolt11": "lnbc10u1pwrmwhppp59mu786ehef0w9rku5zuprz42mwktkg2temzyx55zkx4zhtelwz0qdq6w3jhxapqgf2yxgrsv9uk6etwwscqzysxqz95udv5z0kz9r37xg7rlxh0juc6wqgmnaccdgajstklx0eeuh4ggjx97m7mgqwrx8x4pw5qs46zevsw8rczd9uctehnmmya7hh7tlrumdgph5y4lz",
      "hash": "2ef9e3eb37ca5ee28edca0b8118aaadbacbb214bcec4435282b1aa2baf3f709e",
      "address": "3NFeVNsU77aXyxvtNubdXmHaSde4zrAmgy",
      "description": "test BTC payment",
      "amount": 1000,
      "is_expired": true,
      "is_paid": true,
      "ln_paid": false,
      "btc_paid": true,
      "btc_amount": 1000,
      "confirmations": 0,
      "txids": [
        "978f3e4da26198441ce0605b631ff0b16e6ce944957cd106cb842925225ae7f1"
      ]
    },
    {
      "created_at": 1547552156,
      "expiry": 180,
      "bolt11": "lnbc10u1pwrm3vupp5clhjnqm4jh3ea4e02y5fgj5uqptkfd4g9p6qkvl4hl7e85rmkt2qdqcw3jhxapqv3hh2cnvv5s8qctecqzysxqz959v0qlp623lvpjhm40feypqrcm2d45th069qw505ea2m8xljgvp7h6hcrjp5dpauvg5kdrc3ytqf2svpx9c9m8hcjwam3u4vd2zts6ksq8zl5ux",
      "hash": "c7ef29837595e39ed72f5128944a9c005764b6a828740b33f5bffd93d07bb2d4",
      "address": "3MVZND29Vcsw1XzUNcuv9uFwVDCd8gyWuT",
      "description": "test double pay",
      "amount": 1000,
      "is_expired": true,
      "is_paid": true,
      "ln_paid": true,
      "btc_paid": true,
      "btc_amount": 1000,
      "confirmations": 0,
      "txids": [
        "6aa3d526e054921bca4f2753e1e6456dc2b7c4a80cd0be9e20d5e9386987d8e2"
      ]
    },
    {
      "created_at": 1547552307,
      "expiry": 180,
      "bolt11": "lnbc10u1pwrm33npp5s9mcpq6vntk2e5vdaljnzx6t6fl2hu958aef2g5al4pgyhl8kazqdqjw3jhxapqv4u8q6tj0ycqzysxqz9565njn5eke7yzmed0fud6ne2pmd6naxrz4uy3lftk4xz8w7a4vys5dx239xrudh9xmlvws5kg0upvezfg8q0c39rsxucz589f099xmrcp5wyvhh",
      "hash": "817780834c9aecacd18defe5311b4bd27eabf0b43f7295229dfd42825fe7b744",
      "address": "32LjfFVA1eNuryyYj876Gs9u6nNWx73bVk",
      "description": "test expiry",
      "amount": 1000,
      "is_expired": true,
      "is_paid": false,
      "ln_paid": false,
      "btc_paid": false,
      "btc_amount": 0,
      "confirmations": 0,
      "txids": []
    },
    {
      "created_at": 1547552348,
      "expiry": 180,
      "bolt11": "lnbc10u1pwrm3jupp5r7huhz6hf4urp64zgfg4cq2xnyyzehju0h8gz86dxzfp7pa4zqvsdq5w3jhxapqwpjkuerfdenscqzysxqz95xtkpjvdjtpc7wd608zn8v75wtv7prjtcz5jqg09h7yec6zvtkfdxvh5lqag56vj878wuw0umjae7jyne3a2cqq2yns2znjfl3kgl9ssqhregy7",
      "hash": "1fafcb8b574d7830eaa242515c014699082cde5c7dce811f4d30921f07b51019",
      "address": "36bZ4tXUYAM8CB1QcCq8Lb5Kmh4S6vnC1Z",
      "description": "test pending",
      "amount": 1000,
      "is_expired": false,
      "is_paid": false,
      "ln_paid": false,
      "btc_paid": false,
      "btc_amount": 0,
      "confirmations": 0,
      "txids": []
    }
  ]
}
```

> **NOTE:** most recent invoice is on the bottom  


Development
---

All contributions are welcome.

Feel free to get in touch!

---
Made with 🥩 in Chiang Mai
