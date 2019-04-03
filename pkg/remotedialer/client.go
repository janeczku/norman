package remotedialer

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type ConnectAuthorizer func(proto, address string) bool

func ClientConnect(wsURL string, headers http.Header, dialer *websocket.Dialer, auth ConnectAuthorizer, onConnect func(context.Context) error) {
	if err := connectToProxy(wsURL, headers, auth, dialer, onConnect); err != nil {
		logrus.WithError(err).Error("Failed to connect to proxy")
		time.Sleep(time.Duration(5) * time.Second)
	}
}

func connectToProxy(proxyURL string, headers http.Header, auth ConnectAuthorizer, dialer *websocket.Dialer, onConnect func(context.Context) error) error {
	logrus.WithField("url", proxyURL).Info("Connecting to proxy")

	if dialer == nil {
		dialer = &websocket.Dialer{}
	}
	ws, resp, err := dialer.Dial(proxyURL, headers)
	if err != nil {
		if err == websocket.ErrBadHandshake {
			// The websocket library returns ErrBadHandshake when the server response
			// to opening handshake is invalid. To facilitate troubleshooting we log
			// a junk of the response body which usually includes the error string.
			body := make([]byte, 128)
			io.ReadFull(resp.Body, body)
			logrus.WithFields(logrus.Fields{
				"StatusCode": resp.StatusCode,
				"Body":   string(body),
			}).Error("Invalid proxy response")
			resp.Body.Close()
		}
		return err
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if onConnect != nil {
		if err := onConnect(ctx); err != nil {
			return err
		}
	}

	session := NewClientSession(auth, ws)
	_, err = session.Serve()
	session.Close()
	return err
}
