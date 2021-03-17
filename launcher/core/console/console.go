package console

import (
	"fmt"
	"runtime"
)

const (
	DefaultBanner = `
			   ___                 ___  _____  __
			  / _ \ _ __  ___ _ _ |   \| __\ \/ /
			 | (_) | '_ \/ -_) ' \| |) | _| >  < 
			  \___/| .__/\___|_||_|___/|___/_/\_\
				   |_|
--------------------------------------------------------------
`)

type Console struct {
	Banner string
	ShowBanner bool
}

var DefaultConsole *Console

func init() {
	DefaultConsole = &Console{
		Banner: DefaultBanner,
		ShowBanner: true,
	}
}

func (t *Console) Start(launcherExecutable string) error {
	if t.ShowBanner {
		fmt.Println(t.Banner)
	}
	if runtime.GOOS == "windows" {
		return startPowershell()
	} else {
		return startBash(launcherExecutable)
	}
}
