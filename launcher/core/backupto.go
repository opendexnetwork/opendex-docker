package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/opendexnetwork/opendex-docker/launcher/utils"
	"strings"
)

func (t *Launcher) BackupTo(ctx context.Context, location string) error {
	t.BackupDir = location
	if err := t.Apply(); err != nil {
		return err
	}
	if err := t.Gen(ctx); err != nil {
		return err
	}
	if err := t.upOpendexd(ctx); err != nil {
		return err
	}
	return nil
}

func (t *Launcher) _backupto(c *websocket.Conn, reqId uint64, args []string) {
	if len(args) != 1 {
		t.respondError(c, reqId, errors.New("unexpected arguments: " + strings.Join(args, ", ")))
		return
	}
	location := args[0]
	location = strings.TrimSpace(location)
	if location == "" {
		t.respondError(c, reqId, errors.New("empty location"))
		return
	}
	if !utils.FileExists(location) {
		t.respondError(c, reqId, errors.New("non-existent location: " + location))
		return
	}
	if !utils.IsDir(location) {
		t.respondError(c, reqId, errors.New("location is not a directory: " + location))
		return
	}
	err := t.BackupTo(context.Background(), location)
	if err != nil {
		t.respondError(c, reqId, err)
	} else {
		t.respondResult(c, reqId, fmt.Sprintf("Changed backup location to %s", location))
	}
}
