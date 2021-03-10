package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/opendexnetwork/opendex-docker/launcher/service/proxy"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"golang.org/x/sync/errgroup"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
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

func (t *Launcher) Pull(ctx context.Context) error {
	t.Logger.Debugf("Pulling images")
	cmd := exec.Command("docker-compose", "pull")
	return utils.Run(ctx, cmd)
}

func (t *Launcher) Setup(ctx context.Context, pull bool, interactive bool) error {
	if interactive {
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

	if interactive {
		fmt.Printf("🌍 Checking for updates ...\n")
	}

	if pull {
		if err := t.Pull(ctx); err != nil {
			return fmt.Errorf("pull: %w", err)
		}
	}

	if interactive {
		fmt.Printf("🏃 Warming up ...\n")
	}

	t.Logger.Debugf("Bring up proxy")
	if err := t.upProxy(ctx); err != nil {
		return fmt.Errorf("up proxy: %w", err)
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

	t.Logger.Debugf("Bring up layer 2 services")
	if err := t.upLayer2(ctx); err != nil {
		return fmt.Errorf("up layer2: %w", err)
	}

	t.Logger.Debugf("Bring up opendexd")
	if err := t.upOpendexd(ctx); err != nil {
		return fmt.Errorf("up opendexd: %w", err)
	}

	t.Logger.Debugf("Bring up additional services")
	if err := t.upArby(ctx); err != nil {
		return fmt.Errorf("up arby: %w", err)
	}
	if err := t.upBoltz(ctx); err != nil {
		return fmt.Errorf("up boltz: %w", err)
	}

	_, err = f.WriteString("Start shell\n")
	if err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	_ = f.Close()

	if interactive {
		// enter into console
		return t.StartConsole(ctx)
	} else {
		fmt.Println("Attached to proxy. Press Ctrl-C to detach from it.")
		wg.Wait()
	}

	return nil
}

func (t *Launcher) upProxy(ctx context.Context) error {
	return t.upService(ctx, "proxy", func(status string) bool {
		if status == "Ready" {
			return true
		}
		return false
	})
}

func (t *Launcher) upLnd(ctx context.Context, name string, syncing chan LndSyncing) error {
	return t.upService(ctx, name, func(status string) bool {
		syncing <- LndSyncing{
			Name: name,
			Status: status,
		}

		if status == "Ready" {
			return true
		}
		if strings.HasPrefix(status, "Syncing 100.00%") {
			return true
		}
		if strings.HasPrefix(status, "Syncing 99.99%") {
			return true
		}
		if strings.HasPrefix(status, "Wallet locked") {
			return true
		}

		return false
	})
}

func (t *Launcher) upConnext(ctx context.Context) error {
	return t.upService(ctx, "connext", func(status string) bool {
		if status == "Ready" {
			return true
		}
		return false
	})
}

func (t *Launcher) getProxyApiUrl() (string, error) {
	s, err := t.GetService("proxy")
	if err != nil {
		return "", fmt.Errorf("get service: %w", err)
	}
	rpc, err := s.GetRpcParams()
	if err != nil {
		return "", fmt.Errorf("get rpc params: %w", err)
	}
	return rpc.(proxy.RpcParams).ToUri(), nil
}

type ApiError struct {
	Message string `json:"message"`
}

func (t *Launcher) createWalletsByProxy(ctx context.Context, password string) error {
	apiUrl, err := t.getProxyApiUrl()
	if err != nil {
		return fmt.Errorf("get proxy api url: %w", err)
	}
	createUrl := fmt.Sprintf("%s/api/v1/opendexd/create", apiUrl)
	payload := map[string]interface{}{
		"password": password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", createUrl, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("do reqeust: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body ApiError
		err := json.NewDecoder(resp.Body).Decode(&body)
		if err != nil {
			return fmt.Errorf("[http %d] decode error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("[http %d] %s", resp.StatusCode, body.Message)
	}

	return nil
}

func (t *Launcher) createWalletsByTty(ctx context.Context) error {
	// docker-compose exec opendexd opendex-cli create
	c := exec.Command("docker-compose", "exec", "opendexd", "opendex-cli", "create")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (t *Launcher) createWallets(ctx context.Context) error {
	interactive := ctx.Value("interactive").(bool)
	if interactive {
		return t.createWalletsByTty(ctx)
	} else {
		return t.createWalletsByProxy(ctx, DefaultWalletPassword)
	}
}

func (t *Launcher) unlockWalletsByProxy(ctx context.Context, password string) error {
	apiUrl, err := t.getProxyApiUrl()
	if err != nil {
		return fmt.Errorf("get proxy api url: %w", err)
	}
	createUrl := fmt.Sprintf("%s/api/v1/opendexd/unlock", apiUrl)
	payload := map[string]interface{}{
		"password": password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", createUrl, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("do reqeust: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body ApiError
		err := json.NewDecoder(resp.Body).Decode(&body)
		if err != nil {
			return fmt.Errorf("[http %d] decode error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("[http %d] %s", resp.StatusCode, body.Message)
	}

	return nil
}

func (t *Launcher) unlockWalletsByTty(ctx context.Context) error {
	// docker-compose exec opendexd opendex-cli create
	c := exec.Command("docker-compose", "exec", "opendexd", "opendex-cli", "unlock")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (t *Launcher) unlockWallets(ctx context.Context, password string) error {
	interactive := ctx.Value("interactive").(bool)
	if interactive {
		return t.unlockWalletsByTty(ctx)
	} else {
		return t.unlockWalletsByProxy(ctx, DefaultWalletPassword)
	}
}

func (t *Launcher) upService(ctx context.Context, name string, checkFunc func(string) bool) error {
	s, err := t.GetService(name)
	if err != nil {
		return fmt.Errorf("get service: %w", err)
	}
	if s.IsDisabled() {
		// if the service is running then we will stop it and remove the container
		if s.IsRunning() {
			if err := s.Stop(ctx); err != nil {
				return err
			}
			if err := s.Remove(ctx); err != nil {
				return err
			}
		}
		return nil
	}
	if err := s.Up(ctx); err != nil {
		return fmt.Errorf("up: %s", err)
	}
	prevStatus := ""
	count := 0
	for {
		if count >= ServiceStuckThreshold {
			if ctx.Value("rescue").(bool) {
				if s.Rescue(ctx) {
					count = 0
				} else {
					break
				}
			} else {
				break
			}
		}

		status, err := s.GetStatus(ctx)
		if err != nil {
			t.Logger.Errorf("Failed to get status: %s", err)
			if prevStatus == "" {
				count++
			} else {
				count = 0
			}
			prevStatus = ""
		} else {
			t.Logger.Debugf("[status] %s: %s", name, status)
			if prevStatus == status {
				count++
			} else {
				count = 0
			}
			prevStatus = status

			if status == "Container missing" || status == "Container exited" {
				return fmt.Errorf("%s: %s", name, status)
			}

			if checkFunc(status) {
				break
			}
		}
		select {
		case <-ctx.Done(): // context cancelled
			return errInterrupted
		case <-time.After(StatusQueryInterval): // retry
		}
	}

	if count >= ServiceStuckThreshold {
		return fmt.Errorf("%s stuck", name)
	}

	return nil
}

func (t *Launcher) upOpendexd(ctx context.Context) error {
	// Do you want to create a new opendexd environment or restore an existing one?
	// 1) Create New
	// 2) Restore Existing
	// Please choose: 1

	// Please enter a path to a destination where to store a backup of your environment. It includes everything, but NOT your on-chain wallet balance which is secured by your opendexd SEED. The path should be an external drive, like a USB or network drive, which is permanently available on your device since backups are written constantly.
	//
	// Enter path to backup location: /media/USB/
	// Checking... OK.
	return t.upService(ctx, "opendexd", func(status string) bool {
		if status == "Ready" {
			return true
		}
		if status == "Waiting for channels" {
			return true
		}
		if strings.HasPrefix(status, "Wallet missing") {
			if err := t.createWallets(ctx); err != nil {
				t.Logger.Errorf("Failed to create: %s", err)
				return false
			}
			_, err := os.Create(t.PasswordUnsetMarker)
			if err != nil {
				t.Logger.Errorf("Failed to create .default-password: %s", err)
				return false
			}
			return false
		} else if strings.HasPrefix(status, "Wallet locked") {
			if t.UsingDefaultPassword() {
				if err := t.unlockWallets(ctx, DefaultWalletPassword); err != nil {
					t.Logger.Errorf("Failed to unlock: %s", err)
					if strings.Contains(err.Error(), "password is incorrect") {
						_ = os.Remove(t.PasswordUnsetMarker)
						return true // don't try to unlock with wrong password infinitely
					}
					return false
				}
				return false
			}
			return true
		}
		return false
	})
}

func (t *Launcher) upArby(ctx context.Context) error {
	return t.upService(ctx, "arby", func(status string) bool {
		return true
	})
}

func (t *Launcher) upBoltz(ctx context.Context) error {
	return t.upService(ctx, "boltz", func(status string) bool {
		return true
	})
}

func (t *Launcher) attachToProxy(ctx context.Context) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	s, err := t.GetService("proxy")
	if err != nil {
		return err
	}

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

type LndSyncing struct {
	Name string
	Status string
}

func printSyncing(syncing chan LndSyncing) {
	t := utils.SimpleTable{
		Columns: []utils.TableColumn{
			{
				ID: "service",
				Display: "SERVICE",
			},
			{
				ID: "status",
				Display: "STATUS",
			},
		},
		Records: []utils.TableRecord{
			{
				Fields: map[string]string{
					"service": "lndbtc",
					"status": "Syncing...",
				},
			},
			{
				Fields: map[string]string{
					"service": "lndltc",
					"status": "Syncing...",
				},
			},
		},
	}

	fmt.Println()
	fmt.Println("Syncing light clients:")

	t.Print()

	for e := range syncing {
		t.PrintUpdate(utils.TableRecord{
			Fields: map[string]string{
				"service": e.Name,
				"status": e.Status,
			},
		})
	}
}

func (t *Launcher) upLayer2(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	syncing := make(chan LndSyncing)

	g.Go(func() error {
		return t.upLnd(ctx, "lndbtc", syncing)
	})

	g.Go(func() error {
		return t.upLnd(ctx, "lndltc", syncing)
	})

	g.Go(func() error {
		return t.upConnext(ctx)
	})

	interactive := ctx.Value("interactive").(bool)
	if interactive {
		go printSyncing(syncing)
	}

	defer close(syncing)

	return g.Wait();
}
