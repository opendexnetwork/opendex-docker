package base

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	dt "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/opendexnetwork/opendex-docker/launcher/log"
	"github.com/opendexnetwork/opendex-docker/launcher/service"
	"github.com/opendexnetwork/opendex-docker/launcher/types"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Service struct {
	Name    string
	Context types.ServiceContext

	Hostname    string
	Image       string
	Command     []string
	Environment map[string]string
	Ports       []string
	Volumes     []string
	Disabled    bool
	DataDir     string

	client        *docker.Client

	Logger *logrus.Entry
}

func New(ctx types.ServiceContext, name string) *Service {
	client, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		panic(err)
	}

	return &Service{
		Name:          name,
		Context:       ctx,
		client:        client,
		Logger:        log.NewLogger(fmt.Sprintf("service.%s", name)),

		Hostname:    name,
		Image:       "",
		Command:     []string{},
		Environment: make(map[string]string),
		Ports:       []string{},
		Volumes:     []string{},
		Disabled:    false,
		DataDir:     "",
	}
}

func (t *Service) GetName() string {
	return t.Name
}

// GetContainerName parses the container name from "docker-compose ps" command
//
// Example 1.
// docker-compose ps opendexd
//              Name                    Command        State                              Ports
// ----------------------------------------------------------------------------------------------------------------------
// 5efa0e55c882_testnet_opendexd_1   /entrypoint.sh   Exit 255   0.0.0.0:55002->18885/tcp, 18887/tcp, 28887/tcp, 8887/tcp
//
// Example 2.
// docker-compose ps opendexd1
// ERROR: No such service: opendexd1
//
// Example 3.
// docker-compose ps boltz
// Name   Command   State   Ports
// ------------------------------
//
// Example 4.
// Name             Command       State                 Ports
// --------------------------------------------------------------------------------
// testnet_opendexd_1   /entrypoint.sh   Up      0.0.0.0:49153->18885/tcp,
//                                               18887/tcp, 28887/tcp, 8887/tcp
func (t *Service) GetContainerName(ctx context.Context) string {
	c := exec.Command("docker-compose", "ps", "-a", t.Name)
	output, _ := utils.Output(ctx, c)

	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")

	n := len(lines)
	defaultName := fmt.Sprintf("%s_%s_1", t.Context.GetNetwork(), t.Name)

	if n == 0 {
		return defaultName
	} else if strings.HasPrefix("ERROR:", lines[0]) {
		return defaultName
	} else {
		var j int
		for i, line := range lines {
			if strings.ReplaceAll(line, "-", "") == "" {
				j = i
				break
			}
		}
		if j >= n {
			return defaultName
		} else {
			if j + 1 >= n {
				return defaultName
			}
			return strings.Split(lines[j+1], " ")[0]
		}
	}
}

func (t *Service) GetStatus(ctx context.Context) (string, error) {
	c, err := t.getContainer(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return "Container missing", nil
		}
		return "", err
	}
	return fmt.Sprintf("Container %s", c.State.Status), nil
}

func (t *Service) Create(ctx context.Context) error {
	c := exec.Command("docker-compose", "up", "-d", "--no-start", t.Name)
	return utils.Run(ctx, c)
}

func (t *Service) Up(ctx context.Context) error {
	c := exec.Command("docker-compose", "up", "-d", t.Name)
	return utils.Run(ctx, c)
}

func (t *Service) Start(ctx context.Context) error {
	//c := exec.Command("docker-compose", "start", t.Name)
	c := exec.Command("docker-compose", "up", "-d", t.Name)
	return utils.Run(ctx, c)
}

func (t *Service) Stop(ctx context.Context) error {
	c := exec.Command("docker-compose", "stop", t.Name)
	return utils.Run(ctx, c)
}

func (t *Service) Restart(ctx context.Context) error {
	c := exec.Command("docker-compose", "restart", t.Name)
	return utils.Run(ctx, c)
}



func (t *Service) demuxLogsReader(reader io.Reader) io.Reader {
	r, w := io.Pipe()
	go func() {
		stdcopy.StdCopy(w, w, reader)
		w.Close()
	}()
	return r
}

func (t *Service) GetLogs(ctx context.Context, since string, tail string) ([]string, error) {
	reader, err := t.client.ContainerLogs(ctx, t.GetContainerName(ctx), dt.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since,
		Tail:       tail,
		Follow:     false,
	})
	if err != nil {
		return nil, err
	}

	var lines []string
	r := t.demuxLogsReader(reader)

	bufReader := bufio.NewReader(r)
	for {
		line, _, err := bufReader.ReadLine()
		if err != nil {
			break
		}
		lines = append(lines, string(line))
	}

	return lines, nil
}

func (t *Service) FollowLogs(ctx context.Context, since string, tail string) (<-chan string, func(), error) {
	reader, err := t.client.ContainerLogs(ctx, t.GetContainerName(ctx), dt.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since,
		Tail:       tail,
		Follow:     true,
	})
	if err != nil {
		return nil, nil, err
	}

	r := t.demuxLogsReader(reader)

	ch := make(chan string)

	go func() {
		bufReader := bufio.NewReader(r)
		for {
			line, _, err := bufReader.ReadLine()
			if err != nil {
				ch <- "--- EOF ---"
				break
			}
			ch <- string(line)
		}
		close(ch)
	}()

	return ch, func() { reader.Close() }, nil
}

func (t *Service) Exec(ctx context.Context, name string, args ...string) (string, error) {
	createResp, err := t.client.ContainerExecCreate(ctx, t.GetContainerName(ctx), dt.ExecConfig{
		Cmd:          append([]string{name}, args...),
		Tty:          false,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("[docker] create exec: %w", err)
	}

	execId := createResp.ID

	// ContainerExecAttach = ContainerExecStart
	attachResp, err := t.client.ContainerExecAttach(ctx, execId, dt.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("[docker] attach exec: %w", err)
	}

	var buf bytes.Buffer
	_, err = stdcopy.StdCopy(&buf, &buf, attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("[docker] stdcopy: %w", err)
	}

	exec_, err := t.client.ContainerExecInspect(ctx, execId)
	if err != nil {
		return "", fmt.Errorf("[docker] inspect exec: %w", err)
	}
	exitCode := exec_.ExitCode

	if exitCode != 0 {
		output := buf.String()
		msg := fmt.Sprintf("[docker] command \"%s\" exits with non-zero code %d: %s", strings.Join(append([]string{name}, args...), " "), exitCode, strings.TrimSpace(output))
		return "", service.ErrExec{
			Output:   output,
			ExitCode: exitCode,
			Message:  msg,
		}
	}

	return buf.String(), nil
}

func (t *Service) Apply(cfg interface{}) error {
	c := cfg.(Config)

	t.Image = c.Image
	t.Ports = c.ExposePorts
	t.Disabled = c.Disabled
	t.Environment = map[string]string{}
	t.Environment["NETWORK"] = string(t.Context.GetNetwork())
	t.DataDir = c.Dir
	t.Ports = []string{}
	t.Volumes = []string{}
	t.Command = []string{}

	return nil
}

func (t *Service) GetImage() string {
	return t.Image
}

func (t *Service) GetHostname() string {
	return t.Hostname
}

func (t *Service) GetCommand() []string {
	return t.Command
}

func (t *Service) GetEnvironment() map[string]string {
	return t.Environment
}

func (t *Service) GetPorts() []string {
	return t.Ports
}

func (t *Service) GetVolumes() []string {
	return t.Volumes
}

func (t *Service) IsDisabled() bool {
	return t.Disabled
}

func (t *Service) GetRpcParams() (interface{}, error) {
	return make(map[string]interface{}), nil
}

func (t *Service) GetDefaultConfig() interface{} {
	return nil
}

func (t *Service) getContainer(ctx context.Context) (*dt.ContainerJSON, error) {
	c, err := t.client.ContainerInspect(ctx, t.GetContainerName(ctx))
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (t *Service) GetStartedAt(ctx context.Context) (string, error) {
	c, err := t.getContainer(ctx)
	if err != nil {
		return "", err
	}
	return c.State.StartedAt, nil
}

func (t *Service) GetDataDir() string {
	return t.DataDir
}

func (t *Service) GetMode() string {
	return ""
}

func (t *Service) Rescue(ctx context.Context) bool {
	return true
}

func (t *Service) Remove(ctx context.Context) error {
	c := exec.Command("docker-compose", "rm", "-f", t.Name)
	return utils.Run(ctx, c)
}

func (t *Service) RemoveData(ctx context.Context) error {
	err := os.RemoveAll(t.DataDir)
	if err != nil {
		t.Logger.Warnf("Forcefully remove %s", t.DataDir)
		cmd := exec.Command("sudo", "rm", "-rf", t.DataDir)
		return cmd.Run()
	}
	return nil
}

func (t *Service) IsRunning() bool {
	status, err := t.GetStatus(context.Background())
	if err != nil {
		return false
	}
	return status == "Container running"
}
