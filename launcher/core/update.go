package core

import (
	"context"
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"os/exec"
)

func (t *Launcher) Pull(ctx context.Context) error {
	t.Logger.Debugf("Pulling images")
	cmd := exec.Command("docker-compose", "pull")
	return utils.Run(ctx, cmd)
}

func (t *Launcher) Update(ctx context.Context) {
	// TODO update
	fmt.Println("to be implemented")
}
