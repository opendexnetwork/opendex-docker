package core

import (
	"github.com/opendexnetwork/opendex-docker/launcher/core/collections"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
)

type ServiceMap struct {
	m *collections.OrderedMap
}

func NewServiceMap() *ServiceMap {
	return &ServiceMap{
		m: collections.NewOrderedMap(),
	}
}

func (t *ServiceMap) Get(name string) types.Service {
	return t.m.Get(name).(types.Service)
}

func (t *ServiceMap) Put(name string, service types.Service) {
	t.m.Put(name, service)
}

func (t *ServiceMap) Keys() []string {
	return t.m.Keys()
}
