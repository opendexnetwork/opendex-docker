package connext

import (
	"github.com/opendexnetwork/opendex-docker/launcher/service/base"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
	"path/filepath"
)

type BaseConfig = base.Config

type Config struct {
	BaseConfig
}

func (t *Service) GetDefaultConfig() interface{} {
	network := t.Context.GetNetwork()
	var image string
	if network == types.Mainnet {
		image = "connextproject/vector_node:3a29f0b2"
	} else if network == types.Testnet {
		image = "connextproject/vector_node:3a29f0b2"
	} else if network == types.Simnet {
		image = "connextproject/vector_node:3a29f0b2"
	}

	return &Config{
		BaseConfig: BaseConfig{
			Image:    t.Base.GetBranchImage(image),
			Disabled: false,
			Dir:      filepath.Join(t.Context.GetDataDir(), t.Name),
		},
	}
}
