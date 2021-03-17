package litecoind

import (
	"github.com/opendexnetwork/opendex-docker/launcher/service/bitcoind"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
)

type Base = bitcoind.Service

type Service struct {
	*Base
}

func New(ctx types.ServiceContext, name string) *Service {
	s := bitcoind.New(ctx, name)
	s.ContainerDataDir = "/root/.litecoind"

	return &Service{
		Base: s,
	}
}

func (t *Service) Apply(cfg interface{}) error {
	c := cfg.(*Config)
	if err := t.Base.Apply(&c.BaseConfig); err != nil {
		return err
	}

	network := t.Context.GetNetwork()

	if t.Mode == bitcoind.Native {
		if network == types.Mainnet {
			t.RpcParams.Port = 9332
		} else {
			t.RpcParams.Port = 19332
		}
	}

	return nil
}
