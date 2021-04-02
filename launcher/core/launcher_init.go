package core

import (
	"bufio"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/opendexnetwork/opendex-docker/launcher/log"
	"github.com/opendexnetwork/opendex-docker/launcher/service/lnd"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (t *Launcher) Init() error {
	var err error

	if t.HomeDir, err = ensureHomeDir(); err != nil {
		return err
	}

	t.Network = getNetwork()

	if t.NetworkDir, err = ensureNetworkDir(t.HomeDir, t.Network); err != nil {
		return err
	}

	// fix folder permission
	//if runtime.GOOS == "linux" {
	//	user := os.Getenv("USER")
	//	c := exec.Command("sudo", "chmod", "-R", fmt.Sprintf("%s:%s", user, user), networkDir)
	//	_ = c.Run()
	//} else if runtime.GOOS == "darwin" {
	//	user := os.Getenv("USER")
	//	c := exec.Command("sudo", "chmod", "-R", fmt.Sprintf("%s:staff", user), networkDir)
	//	_ = c.Run()
	//}
	if err := checkFolderPermission(t.NetworkDir); err != nil {
		return err
	}

	t.DataDir = filepath.Join(t.NetworkDir, "data")

	if _, err := os.Stat(t.DataDir); os.IsNotExist(err) {
		if err := os.Mkdir(t.DataDir, 0755); err != nil {
			return err
		}
	}

	logsDir := filepath.Join(t.NetworkDir, "logs")

	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		if err := os.Mkdir(logsDir, 0755); err != nil {
			return err
		}
	}

	dockerComposeFile := filepath.Join(t.NetworkDir, "docker-compose.yml")
	t.ConfigFile = filepath.Join(t.NetworkDir, "config.json")
	passwordUnsetMarker := filepath.Join(t.NetworkDir, ".password-unset")
	// migration for legacy .default-password file
	legacyPasswordUnsetMarker := filepath.Join(t.NetworkDir, ".default-password")
	if utils.FileExists(legacyPasswordUnsetMarker) {
		if err := os.Rename(legacyPasswordUnsetMarker, passwordUnsetMarker); err != nil {
			return err
		}
	}
	t.BackupUnsetMarker = filepath.Join(t.NetworkDir, ".backup-unset")

	t.BackupDir = getBackupDir(t.NetworkDir, dockerComposeFile)
	t.DefaultBackupDir = getDefaultBackupDir(t.NetworkDir)

	t.ExternalIp = getExternalIp(t.NetworkDir)

	//config := tls.Config{RootCAs: nil, InsecureSkipVerify: true}

	t.Logger = log.NewLogger("launcher")
	logfile := filepath.Join(logsDir, "launcher.log")
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	log.SetOutput(f)

	t.Proxy = NewProxy(t, "proxy")
	t.Bitcoind = NewBitcoind(t, "bitcoind")
	t.Litecoind = NewLitecoind(t, "litecoind")
	t.Geth = NewGeth(t, "geth")
	t.Lndbtc = NewLnd(t, "lndbtc", lnd.Bitcoin)
	t.Lndltc = NewLnd(t, "lndltc", lnd.Litecoin)
	t.Connext = NewConnext(t, "connext")
	t.Opendexd = NewOpendexd(t, "opendexd")
	t.Arby = NewArby(t, "arby")
	t.Boltz = NewBoltz(t, "boltz")

	t.Services.Put("proxy", t.Proxy)
	t.Services.Put("bitcoind", t.Bitcoind)
	t.Services.Put("litecoind", t.Litecoind)
	t.Services.Put("geth", t.Geth)
	t.Services.Put("lndbtc", t.Lndbtc)
	t.Services.Put("lndltc", t.Lndltc)
	t.Services.Put("connext", t.Connext)
	t.Services.Put("opendexd", t.Opendexd)
	t.Services.Put("arby", t.Arby)
	t.Services.Put("boltz", t.Boltz)

	return nil
}

func getHomeDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %s", err)
	}
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(homeDir, ".opendex-docker"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "OpendexDocker"), nil
	case "windows":
		return filepath.Join(homeDir, "AppData", "Local", "OpendexDocker"), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func ensureHomeDir() (string, error) {
	dir, err := getHomeDir()
	if err != nil {
		return "", err
	}

	if ! utils.FileExists(dir) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return "", err
		}
	}

	return dir, nil
}

func getNetwork() types.Network {
	if value, ok := os.LookupEnv("NETWORK"); ok {
		return types.Network(value)
	}
	return "mainnet"
}

func getNetworkDir(homeDir string, network types.Network) string {
	if value, ok := os.LookupEnv("NETWORK_DIR"); ok {
		return value
	}
	return filepath.Join(homeDir, string(network))
}

func ensureNetworkDir(homeDir string, network types.Network) (string, error) {
	dir := getNetworkDir(homeDir, network)

	if ! utils.FileExists(dir) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return "", err
		}
	}

	return dir, nil
}

func getBackupDir(networkDir string, dockerComposeFile string) string {
	dir := getDefaultBackupDir(networkDir)

	// TODO parse backup location from 1) opendexd container 2) docker-compose file 3) config.json
	f, err := os.Open(dockerComposeFile)
	if err != nil {
		return dir
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "/root/backup") {
			line = strings.TrimSpace(line)
			// fix broken colon (before) in backup location (
			line = strings.ReplaceAll(line, "::", ":")
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimSuffix(line, ":/root/backup")
			return line
		}
	}
	return dir
}

func getDefaultBackupDir(networkDir string) string {
	return filepath.Join(networkDir, "backup")
}

func getExternalIp(networkDir string) string {
	// Backward compatible with lnd.env
	lndEnv := filepath.Join(networkDir, "lnd.env")
	f, err := os.Open(lndEnv)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key == "EXTERNAL_IP" {
				return value
			}
		}
	}
	return ""
}
