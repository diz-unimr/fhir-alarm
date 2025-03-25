package fhir

import (
	"crypto/tls"
	"fhir-alarm/config"
	"fhir-alarm/notification"
	"fmt"
	"github.com/gorilla/websocket"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"time"
)

type Client struct {
	wsUrl          url.URL
	conn           *websocket.Conn
	tlsConfig      *tls.Config
	subscriptionId string
	emailClient    *notification.EmailClient
}

func NewClient(config config.AppConfig) *Client {
	wsUrl := url.URL{Scheme: "wss", Host: config.Fhir.Server.Host, Path: "/fhir/ws"}

	cert, err := tls.LoadX509KeyPair(config.Fhir.Server.Auth.CertLocation, config.Fhir.Server.Auth.KeyLocation)
	if err != nil {
		slog.Error("Failed to load certificates", "error", err)
		return nil
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return &Client{
		wsUrl:          wsUrl,
		tlsConfig:      tlsConfig,
		subscriptionId: config.Fhir.Server.SubscriptionId,
		emailClient:    notification.NewEmailClient(config.Notification.Email),
	}
}

func (c *Client) ConnectAndBind() error {
	slog.Info("Connecting to websocket", "url", c.wsUrl.String())

	dialer := &websocket.Dialer{
		TLSClientConfig: c.tlsConfig,
	}

	conn, _, err := dialer.Dial(c.wsUrl.String(), nil)
	if err != nil {
		slog.Error("Failed to connect to websocket", "error", err)
		return err
	}

	defer conn.Close()

	done := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// read messages in goroutine
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				slog.Error("Failed to read message", "error", err)
				return
			}
			c.handleMessage(msg)
		}
	}()

	err = bindSubscription(conn, c.subscriptionId)
	if err != nil {
		slog.Error("Failed to bind to subscription", "subscription-id", c.subscriptionId, "error", err)
		closeConnection(conn, done)
	}

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			closeConnection(conn, done)
		}
	}
}

func (c *Client) handleMessage(msg []byte) {
	slog.Info("Message received", "msg", string(msg))
	c.emailClient.Send(msg)
}

func bindSubscription(c *websocket.Conn, id string) error {
	slog.Info("Bind to subscription ", "subscription-id", id)
	return c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("bind %s", id)))
}

func closeConnection(c *websocket.Conn, done chan struct{}) {
	slog.Info("Interrupt received. Sending close message...")

	// send close message and wait for the server to close the connection
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		slog.Error("Failed to write close message to websocket", "error", err)
		return
	}
	select {
	case <-done:
	case <-time.After(time.Second):
	}
	return
}
