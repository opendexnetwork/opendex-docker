package opendexd

import (
	"context"
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/service/proxy"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	DefaultWalletPassword = "OpenDEX!Rocks"
)

func (t *Service) HasWallet() bool {
	nodekey := filepath.Join(t.DataDir, "nodekey.dat")
	return utils.FileExists(nodekey)
}

func (t *Service) IsWalletLocked() bool {
	status, err := t.GetStatus(context.Background())
	if err != nil {
		return false
	}
	return strings.HasPrefix(status, "Wallet locked")
}

func (t *Service) getProxy() *proxy.Service {
	s := t.Context.GetService("proxy")
	return s.(*proxy.Service)
}

func (t *Service) CreateWallet(ctx context.Context, password string) error {
	if password == "" {
		c := exec.Command("docker-compose", "exec", "opendexd", "opendex-cli", "create")
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	} else {
		proxy := t.getProxy()
		return proxy.ApiClient.V1OpendexdCreate(ctx, password)
	}
}

func (t *Service) RestoreWallet(ctx context.Context) error {
	return nil
}

func (t *Service) UnlockWallet(ctx context.Context, password string) error {
	if password == "" {
		c := exec.Command("docker-compose", "exec", "opendexd", "opendex-cli", "unlock")
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	} else {
		proxy := t.getProxy()
		return proxy.ApiClient.V1OpendexdUnlock(ctx, password)
	}
}

func (t *Service) CheckBackup(ctx context.Context) error {
	return nil
}

func (t *Service) ChangeBackupLocation(ctx context.Context) {

}

func (t *Service) SetupWallet(ctx context.Context) error {
	tty := utils.Isatty(os.Stdin)

	if t.HasWallet() {
		if t.IsWalletLocked() {
			if tty {
				t.UnlockWallet(ctx, "")
			} else {
				if err := t.UnlockWallet(ctx, DefaultWalletPassword); err != nil {
					return err
				}
			}
		}
	} else {
		if tty {
			for {
				fmt.Println("Do you want to create a new opendexd environment or restore an existing one?")
				fmt.Println("1) Create New")
				fmt.Println("2) Restore Existing")
				fmt.Print("Please choose: ")
				var choice string
				if _, err := fmt.Scanf("%s", choice); err != nil {
					return err
				}
				switch choice {
				case "1":
					if err := t.CreateWallet(ctx, ""); err != nil {
						return err
					}
					break
				case "2":
					if err := t.RestoreWallet(ctx); err != nil {
						return err
					}
					break
				}
			}
		} else {
			if err := t.CreateWallet(ctx, DefaultWalletPassword); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Service) SetupBackup(ctx context.Context) error {
	return nil
}
