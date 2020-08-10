package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/orange-cloudfoundry/promfetcher/api"
	"github.com/orange-cloudfoundry/promfetcher/clients"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/fetchers"
	"github.com/orange-cloudfoundry/promfetcher/scrapers"
	"github.com/orange-cloudfoundry/promfetcher/userdocs"
	log "github.com/sirupsen/logrus"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "", "Configuration File")
	flag.Parse()

	c, err := config.DefaultConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err.Error())
	}

	if configFile != "" {
		c, err = config.InitConfigFromFile(configFile)
		if err != nil {
			log.Fatal("Error loading config: ", err.Error())
		}
	}
	backendFactory := clients.NewBackendFactory(*c)
	scraper := scrapers.NewScraper(backendFactory, c.DB)
	routeFetcher := fetchers.NewRoutesFetcher(c.Gorouters)
	metricsFetcher := fetchers.NewMetricsFetcher(scraper, routeFetcher)

	rtr := mux.NewRouter()
	api.Register(
		rtr, metricsFetcher,
		api.NewBroker(
			c.Broker,
			c.BaseURL,
			c.DB,
		),
		userdocs.NewUserDoc(c.BaseURL),
	)

	log.Info("Init route fetcher ...")
	routeFetcher.Run()

	if !c.NotExitWhenConnFailed {
		go checkDbConnection(c.DB)
	}

	srvSignal := make(chan os.Signal, 1)
	signal.Notify(srvSignal, syscall.SIGTERM, syscall.SIGINT)

	srvCtx, cancel := context.WithCancel(context.Background())

	go func() {
		<-srvSignal
		cancel()
	}()

	listener, err := makeListener(c)
	if err != nil {
		log.Fatal(err.Error())
	}
	srv := &http.Server{Handler: rtr}

	go func() {
		if err = srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %+s\n", err)
		}
	}()
	defer srv.Close()

	<-srvCtx.Done()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer func() {
		cancel()
	}()

	err = srv.Shutdown(ctxShutDown)
	if err != nil {
		log.Fatalf("server shutdown gracefully Failed: %s\n", err.Error())
	}
	log.Info("server gracefully shutdown")
}

func makeListener(c *config.Config) (net.Listener, error) {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", c.Port)
	if !c.EnableSSL {
		log.Infof("Listen %s without tls ...", listenAddr)
		return net.Listen("tcp", listenAddr)
	}
	log.Infof("Listen %s with tls ...", listenAddr)
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = nil
	}
	if err == nil {
		if c.CACerts != "" {
			if ok := rootCAs.AppendCertsFromPEM([]byte(c.CACerts)); !ok {
				return nil, fmt.Errorf("error adding a CA cert to cert pool")
			}
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{c.SSLCertificate},
		ClientCAs:    rootCAs,
	}

	tlsConfig.BuildNameToCertificate()
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(listener, tlsConfig), nil
}

func checkDbConnection(db *gorm.DB) {
	if db == nil {
		return
	}
	for {
		err := db.DB().Ping()
		if err != nil {
			db.Close()
			log.Fatalf("Error when pinging database: %s", err.Error())
		}
		time.Sleep(5 * time.Minute)
	}
}
