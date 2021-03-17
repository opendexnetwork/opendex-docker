package console

import (
	"fmt"
	"runtime"
)

const (
	DefaultBanner = `
             .___                         __  .__   
           __| _/____ ___  ___      _____/  |_|  |  
          / __ |/ __ \\  \/  /    _/ ___\   __\  |  
         / /_/ \  ___/ >    <     \  \___|  | |  |__
         \____ |\___  >__/\_ \     \___  >__| |____/
              \/    \/      \/         \/           

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
