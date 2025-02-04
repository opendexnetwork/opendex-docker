package core

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/mitchellh/go-homedir"
	"github.com/opendexnetwork/opendex-docker/launcher/log"
	"github.com/opendexnetwork/opendex-docker/launcher/service/arby"
	"github.com/opendexnetwork/opendex-docker/launcher/service/bitcoind"
	"github.com/opendexnetwork/opendex-docker/launcher/service/boltz"
	"github.com/opendexnetwork/opendex-docker/launcher/service/connext"
	"github.com/opendexnetwork/opendex-docker/launcher/service/geth"
	"github.com/opendexnetwork/opendex-docker/launcher/service/litecoind"
	"github.com/opendexnetwork/opendex-docker/launcher/service/lnd"
	"github.com/opendexnetwork/opendex-docker/launcher/service/opendexd"
	"github.com/opendexnetwork/opendex-docker/launcher/service/proxy"
	"github.com/opendexnetwork/opendex-docker/launcher/service/webui"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type Launcher struct {
	Logger         *logrus.Entry
	Services       map[string]types.Service
	ServicesOrder  []string
	ServicesConfig map[string]interface{}

	HomeDir string

	Network types.Network

	NetworkDir       string
	DataDir          string
	LogsDir          string
	BackupDir        string
	DefaultBackupDir string

	DockerComposeFile string
	ConfigFile        string

	PasswordUnsetMarker string
	ExternalIp          string

	rootCmd *cobra.Command
	client  *http.Client

	rootLogger *logrus.Logger

	LogFile *os.File
}

func defaultHomeDir() (string, error) {
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

func checkFolderPermission(networkDir string) error {
	// TODO implement folder permission checking here
	return nil
}

func NewLauncher() (*Launcher, error) {
	homeDir, err := defaultHomeDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		if err := os.Mkdir(homeDir, 0755); err != nil {
			return nil, err
		}
	}

	network := getNetwork()
	networkDir := getNetworkDir(homeDir, network)

	if _, err := os.Stat(networkDir); os.IsNotExist(err) {
		if err := os.Mkdir(networkDir, 0755); err != nil {
			return nil, err
		}
	}

	//if runtime.GOOS == "linux" {
	//	user := os.Getenv("USER")
	//	c := exec.Command("sudo", "chmod", "-R", fmt.Sprintf("%s:%s", user, user), networkDir)
	//	_ = c.Run()
	//} else if runtime.GOOS == "darwin" {
	//	user := os.Getenv("USER")
	//	c := exec.Command("sudo", "chmod", "-R", fmt.Sprintf("%s:staff", user), networkDir)
	//	_ = c.Run()
	//}
	if err := checkFolderPermission(networkDir); err != nil {
		return nil, err
	}

	dataDir := filepath.Join(networkDir, "data")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.Mkdir(dataDir, 0755); err != nil {
			return nil, err
		}
	}

	logsDir := filepath.Join(networkDir, "logs")

	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		if err := os.Mkdir(logsDir, 0755); err != nil {
			return nil, err
		}
	}

	dockerComposeFile := filepath.Join(networkDir, "docker-compose.yml")
	configFile := filepath.Join(networkDir, "config.json")
	passwordUnsetMarker := filepath.Join(networkDir, ".password-unset")
	// migration for legacy .default-password file
	legacyPasswordUnsetMarker := filepath.Join(networkDir, ".default-password")
	if utils.FileExists(legacyPasswordUnsetMarker) {
		if err := os.Rename(legacyPasswordUnsetMarker, passwordUnsetMarker); err != nil {
			return nil, err
		}
	}

	backupDir := getBackupDir(networkDir, dockerComposeFile)
	defaultBackupDir := getDefaultBackupDir(networkDir)

	externalIp := getExternalIp(networkDir)

	config := tls.Config{RootCAs: nil, InsecureSkipVerify: true}

	logfile := filepath.Join(logsDir, "launcher.log")
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)

	l := Launcher{
		Logger:   log.NewLogger("launcher"),
		Services: make(map[string]types.Service),

		HomeDir:             homeDir,
		Network:             network,
		NetworkDir:          networkDir,
		DataDir:             dataDir,
		LogsDir:             logsDir,
		BackupDir:           backupDir,
		DefaultBackupDir:    defaultBackupDir,
		DockerComposeFile:   dockerComposeFile,
		ConfigFile:          configFile,
		PasswordUnsetMarker: passwordUnsetMarker,
		ExternalIp:          externalIp,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &config,
			},
		},
		LogFile: f,
	}

	l.Services, l.ServicesOrder, err = initServices(&l, network)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

func getDefaultValue(dv reflect.Value, fieldName string) interface{} {
	f := dv.FieldByName(fieldName)
	return f.Interface()
}

func (t *Launcher) addFlags(serviceName string, configType reflect.Type, defaultValues reflect.Value, cmd *cobra.Command, config reflect.Value) error {
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		fieldName := field.Name
		fieldType := field.Type
		value := getDefaultValue(defaultValues, fieldName)

		if fieldType.Kind() == reflect.Struct {
			if err := t.addFlags(serviceName, fieldType, reflect.ValueOf(value), cmd, config.FieldByIndex([]int{i})); err != nil {
				return err
			}
			continue
		}

		p := config.FieldByName(fieldName).Addr().Interface()

		key := fmt.Sprintf("%s.%s", serviceName, strcase.ToKebab(fieldName))

		usage := field.Tag.Get("usage")

		// t.Logger.Debugf("[flag] --%s (%#v)", key, value)

		switch fieldType.Kind() {
		case reflect.String:
			cmd.PersistentFlags().StringVar(p.(*string), key, value.(string), usage)
		case reflect.Bool:
			cmd.PersistentFlags().BoolVar(p.(*bool), key, value.(bool), usage)
		case reflect.Uint16:
			cmd.PersistentFlags().Uint16Var(p.(*uint16), key, value.(uint16), usage)
		case reflect.Slice:
			// FIXME differentiate slice item type
			cmd.PersistentFlags().StringSliceVar(p.(*[]string), key, value.([]string), usage)
		default:
			return errors.New("unsupported config struct field type: " + fieldType.Kind().String())
		}
		if err := viper.BindPFlag(key, cmd.PersistentFlags().Lookup(key)); err != nil {
			return err
		}
	}

	return nil
}

func (t *Launcher) AddServiceFlags(cmd *cobra.Command) error {

	t.ServicesConfig = make(map[string]interface{})

	for _, name := range t.ServicesOrder {
		s := t.Services[name]

		defaultConfig := s.GetDefaultConfig()

		configPtr := reflect.TypeOf(defaultConfig)
		if configPtr.Kind() != reflect.Ptr {
			return errors.New("GetDefaultConfig should return a reference of config struct")
		}
		configType := configPtr.Elem() // real config type
		//t.Logger.Debugf("%s: %s.%s", s.GetName(), configType.PkgPath(), configType.Name())

		dv := reflect.ValueOf(defaultConfig).Elem()

		config := reflect.New(configType)

		t.ServicesConfig[name] = config.Interface()

		if err := t.addFlags(name, configType, dv, cmd, reflect.Indirect(config)); err != nil {
			return err
		}
	}
	return nil
}

func initServices(ctx types.Context, network types.Network) (map[string]types.Service, []string, error) {
	var services []types.Service
	var order []string

	proxy_, err := proxy.New(ctx, "proxy")
	if err != nil {
		return nil, nil, err
	}

	lndbtc, err := lnd.New(ctx, "lndbtc", lnd.Bitcoin)
	if err != nil {
		return nil, nil, err
	}

	lndltc, err := lnd.New(ctx, "lndltc", lnd.Litecoin)
	if err != nil {
		return nil, nil, err
	}

	connext_, err := connext.New(ctx, "connext")
	if err != nil {
		return nil, nil, err
	}

	opendexd_, err := opendexd.New(ctx, "opendexd")
	if err != nil {
		return nil, nil, err
	}

	arby_, err := arby.New(ctx, "arby")
	if err != nil {
		return nil, nil, err
	}

	webui_, err := webui.New(ctx, "webui")
	if err != nil {
		return nil, nil, err
	}

	switch network {
	case "simnet":
		services = []types.Service{
			proxy_,
			lndbtc,
			lndltc,
			connext_,
			opendexd_,
			arby_,
			webui_,
		}
		order = []string{
			"proxy",
			"lndbtc",
			"lndltc",
			"connext",
			"opendexd",
			"arby",
			"webui",
		}
	case "testnet":
		bitcoind_, err := bitcoind.New(ctx, "bitcoind")
		if err != nil {
			return nil, nil, err
		}

		litecoind_, err := litecoind.New(ctx, "litecoind")
		if err != nil {
			return nil, nil, err
		}

		geth_, err := geth.New(ctx, "geth")
		if err != nil {
			return nil, nil, err
		}

		boltz_, err := boltz.New(ctx, "boltz")
		if err != nil {
			return nil, nil, err
		}

		services = []types.Service{
			proxy_,
			bitcoind_,
			litecoind_,
			geth_,
			lndbtc,
			lndltc,
			connext_,
			opendexd_,
			arby_,
			boltz_,
			webui_,
		}
		order = []string{
			"proxy",
			"bitcoind",
			"litecoind",
			"geth",
			"lndbtc",
			"lndltc",
			"connext",
			"opendexd",
			"arby",
			"boltz",
			"webui",
		}
	case "mainnet":
		bitcoind_, err := bitcoind.New(ctx, "bitcoind")
		if err != nil {
			return nil, nil, err
		}

		litecoind_, err := litecoind.New(ctx, "litecoind")
		if err != nil {
			return nil, nil, err
		}

		geth_, err := geth.New(ctx, "geth")
		if err != nil {
			return nil, nil, err
		}

		boltz_, err := boltz.New(ctx, "boltz")
		if err != nil {
			return nil, nil, err
		}

		services = []types.Service{
			proxy_,
			bitcoind_,
			litecoind_,
			geth_,
			lndbtc,
			lndltc,
			connext_,
			opendexd_,
			arby_,
			boltz_,
			webui_,
		}
		order = []string{
			"proxy",
			"bitcoind",
			"litecoind",
			"geth",
			"lndbtc",
			"lndltc",
			"connext",
			"opendexd",
			"arby",
			"boltz",
			"webui",
		}
	}

	result := make(map[string]types.Service)
	for _, s := range services {
		result[s.GetName()] = s
	}

	return result, order, nil
}

func (t *Launcher) Run() error {
	err := t.rootCmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func (t *Launcher) GetService(name string) (types.Service, error) {
	if s, ok := t.Services[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("service not found: %s", name)
}

// apply configurations into services
func (t *Launcher) Apply() error {
	for _, name := range t.ServicesOrder {
		s := t.Services[name]
		//t.Logger.Debugf("Apply %s", s.GetName())
		if err := s.Apply(t.ServicesConfig[name]); err != nil {
			return err
		}
	}
	return nil
}

func (t *Launcher) GetNetwork() types.Network {
	return t.Network
}

func (t *Launcher) GetExternalIp() string {
	return ""
}

func (t *Launcher) GetNetworkDir() string {
	return t.NetworkDir
}

func (t *Launcher) GetBackupDir() string {
	return t.BackupDir
}

func (t *Launcher) GetDataDir() string {
	return t.DataDir
}

func (t *Launcher) Close() {
	_ = t.LogFile.Close()
}
