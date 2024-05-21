package mbus

import (
	"net/url"
	"sync/atomic"
	"time"

	"github.com/orange-cloudfoundry/promfetcher/config"

	log "github.com/sirupsen/logrus"

	"code.cloudfoundry.org/tlsconfig"
	"github.com/nats-io/nats.go"
)

type Signal struct{}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Client
type Client interface {
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	Publish(subj string, data []byte) error
}

func Connect(c *config.Config, reconnected chan<- Signal) *nats.Conn {
	var natsClient *nats.Conn
	var natsHost atomic.Value
	var natsAddr atomic.Value
	var err error

	options := natsOptions(c, &natsHost, &natsAddr, reconnected)
	attempts := 3
	for attempts > 0 {
		natsClient, err = options.Connect()
		if err == nil {
			break
		} else {
			attempts--
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil {
		log.Fatalf("nats-connection-error %s", err.Error())
	}

	var natsHostStr string
	natsURL, err := url.Parse(natsClient.ConnectedUrl())
	if err == nil {
		natsHostStr = natsURL.Host
	}
	natsAddrStr := natsClient.ConnectedAddr()

	log.Infof("Successfully-connected-to-nats host: %s addr: %s", natsHostStr, natsAddrStr)

	natsHost.Store(natsHostStr)
	natsAddr.Store(natsAddrStr)
	return natsClient
}

func natsOptions(c *config.Config, natsHost *atomic.Value, natsAddr *atomic.Value, reconnected chan<- Signal) nats.Options {
	options := nats.DefaultOptions
	options.Servers = c.NatsServers()
	if c.Nats.TLSEnabled {
		var err error
		options.TLSConfig, err = tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentity(c.Nats.ClientAuthCertificate),
		).Client(
			tlsconfig.WithAuthority(c.Nats.CAPool),
		)
		if err != nil {
			log.Fatalf("nats-tls-config-invalid %s\n", err.Error())
		}
	}
	options.PingInterval = c.NatsClientPingInterval
	options.MaxReconnect = -1
	notDisconnected := make(chan Signal)

	options.ClosedCB = func(conn *nats.Conn) {
		log.Fatal("nats-connection-closed")
	}

	options.DisconnectedCB = func(conn *nats.Conn) {
		hostStr := natsHost.Load().(string)
		addrStr := natsAddr.Load().(string)
		log.Infof("nats-connection-disconnected host: %s addrStr: %s", hostStr, addrStr)

		go func() {
			ticker := time.NewTicker(c.NatsClientPingInterval)

			for {
				select {
				case <-notDisconnected:
					return
				case <-ticker.C:
					log.Info("nats-connection-still-disconnected")
				}
			}
		}()
	}

	options.ReconnectedCB = func(conn *nats.Conn) {
		notDisconnected <- Signal{}

		natsURL, err := url.Parse(conn.ConnectedUrl())
		natsHostStr := ""
		if err != nil {
			log.Errorf("nats-url-parse-error %s\n", err.Error())
		} else {
			natsHostStr = natsURL.Host
		}
		natsAddrStr := conn.ConnectedAddr()
		natsHost.Store(natsHostStr)
		natsAddr.Store(natsAddrStr)

		log.Infof("nats-connection-reconnected host: %s addr: %s", natsHostStr, natsAddrStr)
		reconnected <- Signal{}
	}

	return options
}
