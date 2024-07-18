package fetchers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"code.cloudfoundry.org/gorouter/common/uuid"

	"code.cloudfoundry.org/localip"
	"github.com/nats-io/nats.go"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/healthchecks"
	"github.com/orange-cloudfoundry/promfetcher/mbus"
	"github.com/orange-cloudfoundry/promfetcher/metrics"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . RoutesFetch

type RoutesFetch interface {
	Run(signals <-chan os.Signal, ready chan<- struct{}) error
	Routes() models.Routes
}

type RoutesFetcher struct {
	mu              sync.Mutex
	routes          *models.Routes
	lastSuccessTime time.Time
	healthCheck     *healthchecks.HealthCheck

	mbusClient       mbus.Client
	subscription     *nats.Subscription
	reconnected      <-chan mbus.Signal
	natsPendingLimit int
	http2Enabled     bool

	params startMessageParams
}

type startMessageParams struct {
	id                               string
	minimumRegisterIntervalInSeconds int
	pruneThresholdInSeconds          int
}

type RouterStart struct {
	Id                               string   `json:"id"`
	Hosts                            []string `json:"hosts"`
	MinimumRegisterIntervalInSeconds int      `json:"minimumRegisterIntervalInSeconds"`
	PruneThresholdInSeconds          int      `json:"pruneThresholdInSeconds"`
}

func NewRoutesFetcher(mbusClient mbus.Client, c *config.Config, reconnected <-chan mbus.Signal, healthCheck *healthchecks.HealthCheck) *RoutesFetcher {
	rts := make(models.Routes)
	guid, err := uuid.GenerateUUID()
	if err != nil {
		log.Fatalf("failed-to-generate-uuid: %s", err.Error())
	}

	return &RoutesFetcher{
		mu:         sync.Mutex{},
		routes:     &rts,
		mbusClient: mbusClient,
		params: startMessageParams{
			id:                               fmt.Sprintf("%d-%s", c.Index, guid),
			minimumRegisterIntervalInSeconds: int(c.StartResponseDelayInterval.Seconds()),
			pruneThresholdInSeconds:          int(c.DropletStaleThreshold.Seconds()),
		},
		reconnected:      reconnected,
		natsPendingLimit: c.NatsClientMessageBufferSize,
		http2Enabled:     c.EnableHTTP2,
		healthCheck:      healthCheck,
	}
}

func (f *RoutesFetcher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	log.Info("subscriber-starting")

	if f.mbusClient == nil {
		return errors.New("subscriber: nil mbus client")
	}
	err := f.sendStartMessage()
	if err != nil {
		return err
	}
	err = f.subscribeToGreetMessage()
	if err != nil {
		return err
	}
	f.subscription, err = f.subscribeRoutes()
	if err != nil {
		return err
	}
	close(ready)

	log.Info("subscriber-started")

	for {
		select {
		case <-f.reconnected:
			err := f.sendStartMessage()
			if err != nil {
				log.Errorf("failed-to-send-start-message: %s", err.Error())
			}
		case <-signals:
			log.Info("exited")
			return nil
		}
	}
}

func (f *RoutesFetcher) Pending() (int, error) {
	if f.subscription == nil {
		log.Error("failed-to-get-subscription")
		return -1, errors.New("NATS subscription is nil, Subscriber must be invoked")
	}

	msgs, _, err := f.subscription.Pending()
	return msgs, err
}

func (f *RoutesFetcher) Dropped() (int, error) {
	if f.subscription == nil {
		log.Error("failed-to-get-subscription")
		return -1, errors.New("NATS subscription is nil, Subscriber must be invoked")
	}

	msgs, err := f.subscription.Dropped()
	return msgs, err
}

func (f *RoutesFetcher) subscribeToGreetMessage() error {
	_, err := f.mbusClient.Subscribe("router.greet", func(msg *nats.Msg) {
		response, _ := f.startMessage()
		_ = f.mbusClient.Publish(msg.Reply, response)
	})

	return err
}

func (f *RoutesFetcher) subscribeRoutes() (*nats.Subscription, error) {
	natsSubscription, err := f.mbusClient.Subscribe("router.*", func(message *nats.Msg) {
		msg, err := mbus.CreateMessage(message.Data)
		if err != nil {
			log.Errorf("validation-error: %s", err.Error())
			log.Errorf("payload: %s", string(message.Data))
			log.Errorf("subject: %s", message.Subject)
			return
		}
		switch message.Subject {
		case "router.register":
			f.registerRoute(msg)
		case "router.unregister":
			f.unregisterRoute(msg)
		default:
		}
	})

	if err != nil {
		return nil, err
	}

	err = natsSubscription.SetPendingLimits(f.natsPendingLimit, f.natsPendingLimit*1024)
	if err != nil {
		return nil, fmt.Errorf("subscriber: SetPendingLimits: %s", err)
	}

	return natsSubscription, nil
}

func (f *RoutesFetcher) startMessage() ([]byte, error) {
	host, err := localip.LocalIP()
	if err != nil {
		return nil, err
	}

	d := RouterStart{
		Id:                               f.params.id,
		Hosts:                            []string{host},
		MinimumRegisterIntervalInSeconds: f.params.minimumRegisterIntervalInSeconds,
		PruneThresholdInSeconds:          f.params.pruneThresholdInSeconds,
	}
	message, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (f *RoutesFetcher) sendStartMessage() error {
	message, err := f.startMessage()
	if err != nil {
		return err
	}
	// Send start message once at start
	return f.mbusClient.Publish("router.start", message)
}

func (f *RoutesFetcher) registerRoute(msg *mbus.Message) {
	route, err := msg.MakeRoute(f.http2Enabled)
	if err != nil {
		log.Errorf("Unable to register route %s", err.Error())
		metrics.ScrapeRouteFailedTotal.With(map[string]string{}).Inc()
		return
	}

	if route.Tags.AppID == "" {
		log.Debugf("Dropped because it is not an app route (%s)", msg.Uris[0])
		return
	}

	for _, uri := range msg.Uris {
		f.routes.RegisterRoute(uri, route)
	}
	metrics.LatestScrapeRoute.With(map[string]string{}).Set(time.Since(f.lastSuccessTime).Seconds())
}

func (f *RoutesFetcher) unregisterRoute(msg *mbus.Message) {
	endpoint, err := msg.MakeRoute(f.http2Enabled)
	if err != nil {
		log.Errorf("Unable to unregister route %s", err.Error())
		metrics.ScrapeRouteFailedTotal.With(map[string]string{}).Inc()
		return
	}

	if endpoint.Tags.AppID == "" {
		log.Debugf("Dropped because it is not an app endpoint (%s)", msg.Uris[0])
		return
	}

	for _, uri := range msg.Uris {
		f.routes.UnregisterRoute(uri, endpoint)
	}
	metrics.LatestScrapeRoute.With(map[string]string{}).Set(time.Since(f.lastSuccessTime).Seconds())
}

func (f *RoutesFetcher) Routes() models.Routes {
	if f.routes == nil {
		return make(models.Routes)
	}
	return *f.routes
}

func (f *RoutesFetcher) RouteHandler(w http.ResponseWriter, req *http.Request) {
	headers := &http.Header{}
	auth := req.Header.Get("Authorization")
	if auth != "" {
		headers.Set("Authorization", auth)
	}
	headers.Set("Content-Type", "application/json")

	routes := f.Routes().String()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(routes))
}
