package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/opendexnetwork/opendex-docker/launcher/core/console"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)


type Launcher struct {
	Logger         *logrus.Entry
	Services       *ServiceMap
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
	BackupUnsetMarker   string
	ExternalIp          string

	rootCmd *cobra.Command
	client  *http.Client

	rootLogger *logrus.Logger

	LogFile *os.File

	Console *console.Console

	Executable string

	// services
	Proxy     *Proxy
	Bitcoind  *Bitcoind
	Litecoind *Litecoind
	Geth      *Geth
	Lndbtc    *Lnd
	Lndltc    *Lnd
	Connext   *Connext
	Opendexd  *Opendexd
	Arby      *Arby
	Boltz     *Boltz
}

func checkFolderPermission(networkDir string) error {
	// TODO implement folder permission checking here
	return nil
}

func NewLauncher() *Launcher {
	return &Launcher{
		Services: NewServiceMap(),
		Console: console.DefaultConsole,
		Executable: getExecutable(),
	}
}

func getExecutable() string {
	exe, err := filepath.Abs(os.Args[0])
	if err != nil {
		panic(err)
	}
	return exe
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

	for _, name := range t.Services.Keys() {
		s := t.Services.Get(name)

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

func (t *Launcher) Run() error {
	err := t.rootCmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

// apply configurations into services
func (t *Launcher) Apply() error {
	for _, name := range t.Services.Keys() {
		s := t.Services.Get(name)
		if err := s.Apply(t.ServicesConfig[name]); err != nil {
			return err
		}
	}
	return nil
}

func (t *Launcher) GetService(name string) types.Service {
	return t.Services.Get(name)
}

func (t *Launcher) GetNetwork() types.Network {
	return t.Network
}

func (t *Launcher) GetExternalIp() string {
	return t.ExternalIp
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

func (t *Launcher) StartConsole(ctx context.Context) error {
	return t.Console.Start(t.Executable)
}

func (t *Launcher) HasService(name string) bool {
	return t.GetService(name) != nil
}
