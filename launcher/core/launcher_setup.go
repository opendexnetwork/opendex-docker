package core

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/opendexnetwork/opendex-docker/launcher/service/proxy"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"time"
)

const (
	DefaultWalletPassword = "OpenDEX!Rocks"
	ServiceStuckThreshold = 100
	StatusQueryInterval   = 10 * time.Second
)

var (
	errInterrupted = errors.New("interrupted")
)

func (t *Launcher) Setup(ctx context.Context, pull bool) error {
	tty := utils.Isatty(os.Stdin)

	if tty {
		fmt.Printf("🚀 Launching %s environment\n", t.Network)
		ctx = context.WithValue(ctx, "interactive", true)
	} else {
		ctx = context.WithValue(ctx, "interactive", false)
	}
	t.Logger.Debugf("Setup %s (%s)", t.Network, t.NetworkDir)

	// Checking Docker
	err := utils.Run(ctx, exec.Command("docker", "info"))
	if err != nil {
		return fmt.Errorf("docker is not ready: %w", err)
	}

	// Changing working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	defer os.Chdir(wd)

	if err := os.Chdir(t.NetworkDir); err != nil {
		return fmt.Errorf("change directory: %w", err)
	}

	logfile := filepath.Join(t.NetworkDir, "logs", fmt.Sprintf("%s.log", t.Network))
	f, err := os.Create(logfile)
	if err != nil {
		return fmt.Errorf("create %s: %w", logfile, err)
	}
	defer f.Close()

	_, err = f.WriteString("Waiting for XUD dependencies to be ready\n")
	if err != nil {
		return fmt.Errorf("write log: %w", err)
	}

	if err := t.Gen(ctx); err != nil {
		return fmt.Errorf("generate files: %w", err)
	}

	if tty {
		fmt.Printf("🌍 Checking for updates ...\n")
	}

	if pull {
		if err := t.Pull(ctx); err != nil {
			return fmt.Errorf("pull: %w", err)
		}
	}

	if tty {
		fmt.Printf("🏃 Warming up ...\n")
	}

	if err := t.Start(ctx, "proxy"); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		t.Logger.Debugf("Attaching to proxy")
		if err := t.attachToProxy(ctx); err != nil {
			t.Logger.Errorf("Attach to proxy: %s", err)
		}
	}()

	//if t.Opendexd.HasWallet()

	if err := t.Start(ctx, "opendexd"); err != nil {
		return err
	}

	//if err := t.Start(ctx, "arby"); err != nil {
	//	return err
	//}

	if err := t.Start(ctx, "boltz"); err != nil {
		return err
	}

	_, err = f.WriteString("Start shell\n")
	if err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	_ = f.Close()

	if tty {
		// enter into console
		return t.StartConsole(ctx)
	} else {
		fmt.Println("Attached to proxy. Press Ctrl-C to detach from it.")
		wg.Wait()
	}

	return nil
}

//func (t *Launcher) upProxy(ctx context.Context) error {
//	return t.upService(ctx, "proxy", func(status string) bool {
//		if status == "Ready" {
//			return true
//		}
//		return false
//	})
//}



//func (t *Launcher) getProxyApiUrl() (string, error) {
//	s, err := t.GetService("proxy")
//	if err != nil {
//		return "", fmt.Errorf("get service: %w", err)
//	}
//	rpc, err := s.GetRpcParams()
//	if err != nil {
//		return "", fmt.Errorf("get rpc params: %w", err)
//	}
//	return rpc.(proxy.RpcParams).ToUri(), nil
//}

//type ApiError struct {
//	Message string `json:"message"`
//}

//func (t *Launcher) createWallets(ctx context.Context) error {
//	//interactive := ctx.Value("interactive").(bool)
//	s, err := t.GetService("opendexd")
//	if err != nil {
//		return err
//	}
//	return s.(*service.Opendexd).CreateWallet(ctx, DefaultWalletPassword)
//}

//func (t *Launcher) unlockWallets(ctx context.Context, password string) error {
//	//interactive := ctx.Value("interactive").(bool)
//	s, err := t.GetService("opendexd")
//	if err != nil {
//		return err
//	}
//	return s.(*service.Opendexd).UnlockWallet(ctx, DefaultWalletPassword)
//}



//func (t *Launcher) upOpendexd(ctx context.Context) error {
//	// Do you want to create a new opendexd environment or restore an existing one?
//	// 1) Create New
//	// 2) Restore Existing
//	// Please choose: 1
//
//	// Please enter a path to a destination where to store a backup of your environment. It includes everything, but NOT your on-chain wallet balance which is secured by your opendexd SEED. The path should be an external drive, like a USB or network drive, which is permanently available on your device since backups are written constantly.
//	//
//	// Enter path to backup location: /media/USB/
//	// Checking... OK.
//	return t.upService(ctx, "opendexd", func(status string) bool {
//		if status == "Ready" {
//			return true
//		}
//		if status == "Waiting for channels" {
//			return true
//		}
//		if strings.HasPrefix(status, "Wallet missing") {
//			if err := t.createWallets(ctx); err != nil {
//				t.Logger.Errorf("Failed to create: %s", err)
//				return false
//			}
//			_, err := os.Create(t.PasswordUnsetMarker)
//			if err != nil {
//				t.Logger.Errorf("Failed to create .default-password: %s", err)
//				return false
//			}
//			return false
//		} else if strings.HasPrefix(status, "Wallet locked") {
//			if t.UsingDefaultPassword() {
//				if err := t.unlockWallets(ctx, DefaultWalletPassword); err != nil {
//					t.Logger.Errorf("Failed to unlock: %s", err)
//					if strings.Contains(err.Error(), "password is incorrect") {
//						_ = os.Remove(t.PasswordUnsetMarker)
//						return true // don't try to unlock with wrong password infinitely
//					}
//					return false
//				}
//				return false
//			}
//			return true
//		}
//		return false
//	})
//}
//
//func (t *Launcher) upArby(ctx context.Context) error {
//	return t.upService(ctx, "arby", func(status string) bool {
//		return true
//	})
//}
//
//func (t *Launcher) upBoltz(ctx context.Context) error {
//	return t.upService(ctx, "boltz", func(status string) bool {
//		return true
//	})
//}

func (t *Launcher) attachToProxy(ctx context.Context) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	s := t.GetService("proxy")

	params, err := s.GetRpcParams()
	if err != nil {
		return err
	}

	port := params.(proxy.RpcParams).Port

	u := url.URL{Scheme: "wss", Host: fmt.Sprintf("127.0.0.1:%d", port), Path: "/launcher"}
	t.Logger.Debugf("Connecting to %s", u.String())

	config := tls.Config{RootCAs: nil, InsecureSkipVerify: true}

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &config
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()

	t.Logger.Debugf("Attached to proxy")

	done := make(chan struct{})

	go func() {
		defer close(done)
		t.serve(ctx, c)
	}()

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			t.Logger.Debugf("Interrupted")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				t.Logger.Errorf("write close: %s", err)
				return nil
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}


