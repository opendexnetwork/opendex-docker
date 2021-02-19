package core

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
)

type WalletsInfo struct {
	DefaultPassword bool `json:"defaultPassword"`
	MnemonicShown   bool `json:"mnemonicShown"`
}

type BackupInfo struct {
	Location        string `json:"location"`
	DefaultLocation bool   `json:"defaultLocation"`
}

type Info struct {
	Wallets WalletsInfo `json:"wallets"`
	Backup  BackupInfo  `json:"backup"`
}

func (t *Launcher) UsingDefaultPassword() bool {
	return utils.FileExists(t.PasswordUnsetMarker)
}

func (t *Launcher) GetInfo() Info {
	defaultPassword := t.UsingDefaultPassword()

	return Info{
		Wallets: WalletsInfo{
			DefaultPassword: defaultPassword,
			MnemonicShown:   !defaultPassword,
		},
		Backup: BackupInfo{
			Location:        t.BackupDir,
			DefaultLocation: t.BackupDir == t.DefaultBackupDir,
		},
	}
}

func (t *Launcher) _getinfo(c *websocket.Conn, id uint64, args []string) {
	info, err := json.Marshal(t.GetInfo())
	if err != nil {
		t.respondError(c, id, err)
	} else {
		t.respondResult(c, id, string(info))
	}
}
