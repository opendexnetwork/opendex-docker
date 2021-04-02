package opendexd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/service"
	"github.com/opendexnetwork/opendex-docker/launcher/service/base"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Base = base.Service

type Service struct {
	*Base
	RpcParams RpcParams
}

func New(ctx types.ServiceContext, name string) *Service {
	s := base.New(ctx, name)

	return &Service{
		Base:      s,
		RpcParams: RpcParams{},
	}
}

type LndInfo struct {
	Status string
}

type ConnextInfo struct {
	Status string
}

type Info struct {
	Lndbtc  LndInfo
	Lndltc  LndInfo
	Connext ConnextInfo
}

func (t *Service) GetInfo(ctx context.Context) (*Info, error) {
	output, err := t.Exec(ctx, "opendex-cli", "getinfo", "-j")
	if err != nil {
		return nil, err
	}
	var result = make(map[string]interface{})
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		return nil, errors.New(output)
	}

	lndbtc := LndInfo{}
	lndltc := LndInfo{}

	for _, item := range result["lndMap"].([]interface{}) {
		info := item.([]interface{})
		switch info[0].(string) {
		case "BTC":
			lndbtc.Status = info[1].(map[string]interface{})["status"].(string)
		case "LTC":
			lndltc.Status = info[1].(map[string]interface{})["status"].(string)
		}
	}

	info := Info{
		Lndbtc: lndbtc,
		Lndltc: lndltc,
		Connext: ConnextInfo{
			Status: result["connext"].(map[string]interface{})["status"].(string),
		},
	}
	return &info, nil
}

func (t *Service) GetStatus(ctx context.Context) (string, error) {
	status, err := t.Base.GetStatus(ctx)
	if err != nil {
		return "", err
	}
	if status != "Container running" {
		return status, nil
	}

	info, err := t.GetInfo(ctx)
	if err != nil {
		if err, ok := err.(service.ErrExec); ok {
			if strings.Contains(err.Output, "opendexd is locked") {
				nodekey := filepath.Join(t.DataDir, "nodekey.dat")
				if _, err := os.Stat(nodekey); os.IsNotExist(err) {
					return "Wallet missing. Create with opendex-cli create/restore.", nil
				}
				return "Wallet locked. Unlock with opendex-cli unlock.", nil
			} else if strings.Contains(err.Output, "tls cert could not be found at /root/.opendex/tls.cert") {
				return "Starting...", nil
			} else if strings.Contains(err.Output, "opendexd is starting") {
				return "Starting...", nil
			} else if strings.Contains(err.Output, "is opendexd running?") {
				// could not connect to opendexd at localhost:18886, is opendexd running?
				return "Starting...", nil
			} else if strings.Contains(err.Output, "No connection established") {
				// Error: 14 UNAVAILABLE: No connection established
				return "Starting...", nil
			}
		}
		return "", fmt.Errorf("get info: %w", err)
	}

	lndbtc := info.Lndbtc.Status
	lndltc := info.Lndltc.Status
	connext := info.Connext.Status

	var notReady []string

	if lndbtc != "Ready" {
		notReady = append(notReady, "lndbtc")
	}

	if lndltc != "Ready" {
		notReady = append(notReady, "lndltc")
	}

	if connext != "Ready" {
		notReady = append(notReady, "connext")
	}

	if len(notReady) == 0 {
		return "Ready", nil
	}

	if strings.Contains(lndbtc, "has no active channels") || strings.Contains(lndltc, "has no active channels") || strings.Contains(connext, "has no active channels") {
		return "Waiting for channels", nil
	}

	return fmt.Sprintf("Waiting for %s", strings.Join(notReady, ", ")), nil
}

func (t *Service) Apply(cfg interface{}) error {
	c := cfg.(*Config)
	if err := t.Base.Apply(c.BaseConfig); err != nil {
		return err
	}
	t.Environment["NODE_ENV"] = "production"

	if c.PreserveConfig {
		t.Environment["PRESERVE_CONFIG"] = "true"
	} else {
		t.Environment["PRESERVE_CONFIG"] = "false"
	}

	lndbtc := t.Context.GetService("lndbtc")

	lndltc := t.Context.GetService("lndltc")

	t.Volumes = append(t.Volumes, fmt.Sprintf("%s:/root/.opendex", t.DataDir))
	t.Volumes = append(t.Volumes, fmt.Sprintf("%s:/root/.lndbtc", lndbtc.GetDataDir()))
	t.Volumes = append(t.Volumes, fmt.Sprintf("%s:/root/.lndltc", lndltc.GetDataDir()))
	t.Volumes = append(t.Volumes, fmt.Sprintf("%s:/root/backup", t.Context.GetBackupDir()))

	network := t.Context.GetNetwork()

	var port uint16

	switch network {
	case types.Simnet:
		port = 28886
		t.Ports = append(t.Ports, "28885")
	case types.Testnet:
		port = 18886
		t.Ports = append(t.Ports, "18885")
	case types.Mainnet:
		port = 8886
		t.Ports = append(t.Ports, "8885")
	}

	t.RpcParams.Type = "gRPC"
	t.RpcParams.Host = t.Name
	t.RpcParams.Port = port
	dataDir := fmt.Sprintf("/root/network/data/%s", t.Name)
	t.RpcParams.TlsCert = fmt.Sprintf("%s/tls.cert", dataDir)

	return nil
}

type RpcParams struct {
	Type    string `json:"type"`
	Host    string `json:"host"`
	Port    uint16 `json:"port"`
	TlsCert string `json:"tlsCert"`
}

func (t *Service) GetRpcParams() (interface{}, error) {
	return t.RpcParams, nil
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

func (t *Service) upLnd(ctx context.Context, name string, syncing chan LndSyncing) error {
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

func (t *Service) upConnext(ctx context.Context) error {
	return t.upService(ctx, "connext", func(status string) bool {
		if status == "Ready" {
			return true
		}
		return false
	})
}

func (t *Service) upService(ctx context.Context, name string, checkFunc func(string) bool) error {
	s := t.Context.GetService(name)
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
	i := 0
	tty := utils.Isatty(os.Stdin)
	for {
		if i > 0 && i % 10 == 0 {
			if tty {
				//fmt.Printf("Still waiting for %s to be ready...\n", t.Name)
			}
		}

		status, err := s.GetStatus(ctx)
		if err != nil {
			return err
		}

		t.Logger.Debugf("[status] %s: %s", name, status)
		i++

		if status == "Container missing" || status == "Container exited" {
			return fmt.Errorf("%s: %s", name, status)
		}

		if checkFunc(status) {
			break
		}

		select {
		case <-ctx.Done(): // context cancelled
			return errors.New("interrupted")
		case <-time.After(3 * time.Second): // retry
		}
	}

	return nil
}

func (t *Service) upDeps(ctx context.Context) error {
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

	return g.Wait()
}

func (t *Service) Start(ctx context.Context) error {
	// make sure layer 2 services (lndbtc, lndltc, and connext) is ready

	if err := t.upDeps(ctx); err != nil {
		return err
	}

	tty := utils.Isatty(os.Stdin)

	if tty {
		fmt.Println("🕹️ Loading OpenDEX console ...")
	}

	if err := t.SetupBackup(ctx); err != nil {
		return err
	}

	if err := t.Base.Start(ctx); err != nil {
		return err
	}

	i := 0
	for {
		if i > 0 && i % 10 == 0 {
			if tty {
				fmt.Printf("Still waiting for %s to be ready...\n", t.Name)
			}
		}
		status, err := t.GetStatus(ctx)
		if err != nil {
			return err
		}
		if status == "Container missing" || status == "Container exited" {
			return fmt.Errorf("%s: %s", t.Name, status)
		} else if status == "Ready" {
			break
		} else if status == "Waiting for channels" {
			break
		} else if strings.HasPrefix(status, "Wallet missing") {
			break
		} else if strings.HasPrefix(status, "Wallet locked") {
			break
		}
		select {
		case <-ctx.Done():
			return errors.New("interrupted")
		case <-time.After(3 * time.Second):
		}
		i++
	}

	if err := t.SetupWallet(ctx); err != nil {
		return err
	}

	return nil
}
