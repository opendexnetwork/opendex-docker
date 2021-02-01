package litecoind

import (
	"github.com/opendexnetwork/opendex-docker/launcher/service/bitcoind"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
)

type BaseConfig = bitcoind.Config

type Config struct {
	BaseConfig
}

func (t *Service) GetDefaultConfig() interface{} {
	base := t.Base.GetDefaultConfig().(*bitcoind.Config)

	network := t.Context.GetNetwork()
	var image string
	if network == types.Mainnet {
		image = "opendexnetwork/litecoind:0.18.1"
	} else {
		image = "opendexnetwork/litecoind:latest"
	}
	base.BaseConfig.Image = t.Base.GetBranchImage(image)

	return &Config{
		BaseConfig: *base,
	}
}
