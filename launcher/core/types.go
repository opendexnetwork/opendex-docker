package core

import (
	"github.com/opendexnetwork/opendex-docker/launcher/service/arby"
	"github.com/opendexnetwork/opendex-docker/launcher/service/bitcoind"
	"github.com/opendexnetwork/opendex-docker/launcher/service/boltz"
	"github.com/opendexnetwork/opendex-docker/launcher/service/connext"
	"github.com/opendexnetwork/opendex-docker/launcher/service/geth"
	"github.com/opendexnetwork/opendex-docker/launcher/service/litecoind"
	"github.com/opendexnetwork/opendex-docker/launcher/service/lnd"
	"github.com/opendexnetwork/opendex-docker/launcher/service/opendexd"
	"github.com/opendexnetwork/opendex-docker/launcher/service/proxy"
)

type Proxy = proxy.Service
var NewProxy = proxy.New
type Opendexd = opendexd.Service
var NewOpendexd = opendexd.New
type Lnd = lnd.Service
var NewLnd = lnd.New
type Connext = connext.Service
var NewConnext = connext.New
type Arby = arby.Service
var NewArby = arby.New
type Boltz = boltz.Service
var NewBoltz = boltz.New
type Bitcoind = bitcoind.Service
var NewBitcoind = bitcoind.New
type Litecoind = litecoind.Service
var NewLitecoind = litecoind.New
type Geth = geth.Service
var NewGeth = geth.New
