package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/opendexnetwork/opendex-docker/launcher/service/proxy"
	"net/url"
	"os"
	"os/signal"
	"time"
)

func (t *Launcher) Attach(endpoint string) error {
	// TODO implement here
	// supports ws and wss
	return nil
}

func (t *Launcher) attachToProxy(ctx context.Context) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	s, err := t.GetService("proxy")
	if err != nil {
		return err
	}

	params, err := s.GetRpcParams()
	if err != nil {
		return err
	}

	port := params.(proxy.RpcParams).Port

	u := url.URL{Scheme: "wss", Host: fmt.Sprintf("127.0.0.1:%d", port), Path: "/launcher"}
	t.Logger.Debugf("Connecting to %s", u.String())

	config := tls.Config{RootCAs: nil, InsecureSkipVerify: true}

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &config
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()

	t.Logger.Debugf("Attached to proxy")

	done := make(chan struct{})

	go func() {
		defer close(done)
		t.serve(ctx, c)
	}()

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			t.Logger.Debugf("Interrupted")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				t.Logger.Errorf("write close: %s", err)
				return nil
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}
