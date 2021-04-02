package webui

import (
	"github.com/opendexnetwork/opendex-docker/launcher/service/base"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
)

type Base = base.Service

type Service struct {
	*Base
}

func New(ctx types.ServiceContext, name string) *Service {
	s := base.New(ctx, name)

	return &Service{
		Base: s,
	}
}

func (t *Service) Apply(cfg interface{}) error {
	c := cfg.(*Config)
	if err := t.Base.Apply(c.BaseConfig); err != nil {
		return err
	}
	return nil
}
