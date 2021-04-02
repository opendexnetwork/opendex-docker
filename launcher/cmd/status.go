package cmd

import (
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

type StatusUpdate struct {
	Service string
	Status string
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get service status",
	PreRunE: CommonPreRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		// Checking Docker
		err := utils.Run(ctx, exec.Command("docker", "info"))
		if err != nil {
			return err
		}

		// Changing working directory
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		defer os.Chdir(wd)

		if err := os.Chdir(launcher.NetworkDir); err != nil {
			return err
		}

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
			Records: []utils.TableRecord{},
		}

		var statusMap = make(map[string]string)

		names := args
		if len(names) == 0 {
			names = launcher.Services.Keys()
		}

		for _, name := range names {
			if ! launcher.HasService(name) {
				return fmt.Errorf("no such service: %s", name)
			}
			statusMap[name] = ""
			t.Records = append(t.Records, utils.TableRecord{
				Fields: map[string]string{
					"service": name,
					"status": "",
				},
			})
		}

		updates := make(chan StatusUpdate)

		for _, name_ := range names {
			name := name_
			go func() {
				status, err := launcher.Status(ctx, name)
				if err != nil {
					if strings.HasPrefix(err.Error(), "Error:") {
						status = err.Error()
					} else {
						status = "Error: " + err.Error()
					}
					updates <- StatusUpdate{Service: name, Status: status}
					return
				}
				updates <- StatusUpdate{Service: name, Status: status}
			}()
		}

		t.Print()

		i := 0

		for i < len(names) {
			update := <-updates
			//fmt.Println(i, update)
			t.PrintUpdate(utils.TableRecord{
				Fields: map[string]string{
					"service": update.Service,
					"status": update.Status,
				},
			})
			i++
		}

		return nil
	},
}
