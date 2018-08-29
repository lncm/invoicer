invoicer
========

> _"fiat is irrelephant 🐘"_

### Exposes simple authenticated API on top of LND

Installing
----------

#### Easy 

Download binary from releases page drop in as the same user as `lnd`.

#### Manual

Have Go 1.11 installed, and:

```bash
git clone https://github.com/lncm/invoicer.git
cd invoicer
make run
``` 

Usage
---

1. Create `users.list` file somewhere and populate it with user credentials. Each pair (username & password) separated with a space, and each line being one pair. 
2. Run binary in background, ex. using `tmux` or `screen`.
3. API binds to `localhost:1666`, and exposes 3 endpoints, as described [here].

[here]: https://github.com/lncm/ideas/issues/5#issuecomment-416109283


Enjoy!

Development
---
All contributions are welcome.

Feel free to get in touch!

---

Made with 🥩 in Chiang Mai
