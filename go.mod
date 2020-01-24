module github.com/lncm/invoicer

go 1.13

require (
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-contrib/gzip v0.0.1
	github.com/gin-gonic/gin v1.4.0
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/golang/protobuf v1.3.1
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lightningnetwork/lnd v0.7.1-beta
	github.com/pelletier/go-toml v1.4.0
	github.com/sirupsen/logrus v1.4.2
	google.golang.org/genproto v0.0.0-20190201180003-4b09977fb922
	google.golang.org/grpc v1.23.0
	gopkg.in/go-playground/validator.v9 v9.29.1
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace (
	github.com/btcsuite/btcwallet => github.com/btcsuite/btcwallet v0.0.0-20190814023431-505acf51507f
	github.com/ugorji/go => github.com/ugorji/go/codec v1.1.7
)
