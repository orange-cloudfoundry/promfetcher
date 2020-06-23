package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
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
	routeFetcher := fetchers.NewRoutesFetcher(c.Gorouter)
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
	listenAddr := fmt.Sprintf("0.0.0.0:%d", c.Port)
	if !c.EnableSSL {
		log.Infof("Listen %s without tls ...", listenAddr)
		err = http.ListenAndServe(listenAddr, rtr)
	} else {
		log.Infof("Listen %s with tls ...", listenAddr)
		err = serveHTTPS(c, rtr)
	}
	if err != nil {
		log.Fatal(err.Error())
	}
}

func serveHTTPS(c *config.Config, handler http.Handler) error {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = nil
	}
	if err == nil {
		if c.CACerts != "" {
			if ok := rootCAs.AppendCertsFromPEM([]byte(c.CACerts)); !ok {
				return fmt.Errorf("error adding a CA cert to cert pool")
			}
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{c.SSLCertificate},
		ClientCAs:    rootCAs,
	}

	tlsConfig.BuildNameToCertificate()
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", c.Port))
	if err != nil {
		return err
	}
	defer listener.Close()
	tlsListener := tls.NewListener(listener, tlsConfig)
	defer tlsListener.Close()

	return http.Serve(tlsListener, handler)
}
