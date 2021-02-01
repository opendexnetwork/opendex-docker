package arby

import (
	"context"
	"errors"
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/service/base"
	_opendexd "github.com/opendexnetwork/opendex-docker/launcher/service/opendexd"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
)

type Base = base.Service
type Opendexd = _opendexd.Service
type OpendexdRpcParams = _opendexd.RpcParams

type Service struct {
	*Base
}

func New(ctx types.Context, name string) (*Service, error) {
	s, err := base.New(ctx, name)
	if err != nil {
		return nil, err
	}

	return &Service{
		Base: s,
	}, nil
}

func (t *Service) getOpendexd() (*Opendexd, error) {
	s, err := t.Context.GetService("opendexd")
	if err != nil {
		return nil, err
	}
	sOpendexd, ok := s.(*Opendexd)
	if !ok {
		return nil, errors.New("cannot convert to *ond.Service")
	}
	return sOpendexd, nil
}

func (t *Service) Apply(cfg interface{}) error {
	c := cfg.(*Config)
	if err := t.Base.Apply(c.BaseConfig); err != nil {
		return err
	}

	opendexd, err := t.getOpendexd()
	if err != nil {
		return err
	}

	t.Volumes = append(t.Volumes,
		fmt.Sprintf("%s:/root/.arby", t.DataDir),
		fmt.Sprintf("%s:/root/.opendexd", opendexd.DataDir),
	)

	params, err := opendexd.GetRpcParams()
	if err != nil {
		return err
	}

	opendexdRpc := params.(OpendexdRpcParams)

	t.Environment["NODE_ENV"] = "production"
	t.Environment["LOG_LEVEL"] = "trace"
	t.Environment["OPENDEX_CERT_PATH"] = "/root/.opendexd/tls.cert"
	t.Environment["OPENDEX_RPC_HOST"] = opendexdRpc.Host
	t.Environment["OPENDEX_RPC_PORT"] = fmt.Sprintf("%d", opendexdRpc.Port)
	t.Environment["BASEASSET"] = c.BaseAsset
	t.Environment["QUOTEASSET"] = c.QuoteAsset
	t.Environment["CEX_BASEASSET"] = c.CexBaseAsset
	t.Environment["CEX_QUOTEASSET"] = c.CexQuoteAsset
	t.Environment["CEX"] = fmt.Sprintf("%s", c.Cex)
	t.Environment["CEX_API_SECRET"] = c.CexApiSecret
	t.Environment["CEX_API_KEY"] = c.CexApiKey
	t.Environment["TEST_MODE"] = fmt.Sprintf("%t", c.TestMode)
	t.Environment["MARGIN"] = c.Margin
	t.Environment["TEST_CENTRALIZED_EXCHANGE_BASEASSET_BALANCE"] = c.TestCentralizedBaseassetBalance
	t.Environment["TEST_CENTRALIZED_EXCHANGE_QUOTEASSET_BALANCE"] = c.TestCentralizedQuoteassetBalance

	return nil
}

func (t *Service) GetStatus(ctx context.Context) (string, error) {
	status, err := t.Base.GetStatus(ctx)
	if err != nil {
		return "", err
	}
	if status != "Container running" {
		return status, nil
	}

	return "Ready", nil
}
