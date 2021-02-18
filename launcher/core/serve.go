package core

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
)

type Request struct {
	Id     uint64   `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func (t *Launcher) serve(ctx context.Context, c *websocket.Conn) {
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			t.Logger.Errorf("read: %s", err)
			return
		}
		t.Logger.Debugf("[attach] recv: %s", message)

		if err := t.handleMessage(c, message); err != nil {
			t.Logger.Errorf("handle %s: %s", message, err)
		}
	}
}

func (t *Launcher) handleMessage(c *websocket.Conn, msg []byte) error {
	var req Request
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}

	switch req.Method {
	case "getinfo":
		t._getinfo(c, req.Id, req.Params)
	case "backupto":
		t._backupto(c, req.Id, req.Params)
	}
	return nil
}

func (t *Launcher) respondError(c *websocket.Conn, reqId uint64, err error) {
	var resp = make(map[string]interface{})
	resp["result"] = nil
	resp["error"] = err.Error()
	resp["id"] = reqId
	j, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	t.Logger.Debugf("[attach] send: %s", j)
	err = c.WriteMessage(websocket.TextMessage, j)
	if err != nil {
		panic(err)
	}
}

func (t *Launcher) respondResult(c *websocket.Conn, reqId uint64, result string) {
	var resp = make(map[string]interface{})
	resp["result"] = result
	resp["error"] = nil
	resp["id"] = reqId
	j, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	t.Logger.Debugf("[attach] send: %s", j)
	err = c.WriteMessage(websocket.TextMessage, j)
	if err != nil {
		panic(err)
	}
}



