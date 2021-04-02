package core

import "context"

func (t *Launcher) Status(ctx context.Context, service string) (string, error) {
	s := t.GetService(service)
	return s.GetStatus(ctx)
}
