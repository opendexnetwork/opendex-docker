package core

import (
	"context"
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
)

type ImageUpdate struct {
	Name string
	OldDigest string
	OldCreated string
	NewDigest string
	NewCreated string
}

type ContainerUpdate struct {
	Name string
	Action string
}

type UpdateDetails struct {
	Images []ImageUpdate
	Containers []ContainerUpdate
}

func (t *Launcher) Update(ctx context.Context, noConfirmation bool) error {
	details := t.checkForUpdates(ctx)
	if details == nil {
		fmt.Println("All up-to-date.")
		return nil
	}
	if t.ShowUpdateDetails {
		fmt.Println(details)
	}
	if !noConfirmation {
		reply := utils.YesNo("Do you want to upgrade?", utils.YES)
		if reply == utils.NO {
			return nil
		}
	}
	return t.applyUpdates(details)
}

func (t *Launcher) applyUpdates(details *UpdateDetails) error {
	return nil
}

func (t *Launcher) checkForUpdates(ctx context.Context) *UpdateDetails {
	return &UpdateDetails{
		Images: t.checkForImageUpdates(ctx),
		Containers: t.checkForContainerUpdates(ctx),
	}
}

func (t *Launcher) checkForImageUpdates(ctx context.Context) []ImageUpdate {
	return nil
}

func (t *Launcher) checkForContainerUpdates(ctx context.Context) []ContainerUpdate {
	return nil
}
