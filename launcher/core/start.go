package core

import "context"

func (t *Launcher) Start(ctx context.Context, service string) error {
	s := t.GetService(service)
	return s.Start(ctx)
}
