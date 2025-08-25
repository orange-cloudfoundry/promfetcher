package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/jinzhu/gorm"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/models"
	"github.com/pivotal-cf/brokerapi/v7"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

type BrokerParams struct {
	Endpoint string `json:"endpoint"`
}

type Broker struct {
	brokerConfig config.BrokerConfig
	baseURL      string
	db           *gorm.DB
}

func NewBroker(brokerConfig config.BrokerConfig, baseURL string, db *gorm.DB) *Broker {
	return &Broker{brokerConfig: brokerConfig, baseURL: baseURL, db: db}
}

func (b *Broker) Handler() http.Handler {
	lag := lager.NewLogger("broker")
	lag.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	return brokerapi.New(b, lag, brokerapi.BrokerCredentials{
		Username: b.brokerConfig.User,
		Password: b.brokerConfig.Pass,
	})
}

func (b Broker) Services(ctx context.Context) ([]domain.Service, error) {
	t := true
	return []domain.Service{
		{
			ID:                   b.brokerConfig.BrokerServiceID,
			Name:                 "promfetcher",
			Description:          "Fetch your prometheus metrics on each instance of your app",
			Bindable:             true,
			InstancesRetrievable: false,
			BindingsRetrievable:  true,
			Tags:                 nil,
			PlanUpdatable:        false,
			Plans: []domain.ServicePlan{
				{
					ID:          b.brokerConfig.BrokerPlanID,
					Name:        "fetch-app",
					Description: "Fetch your prometheus metrics on each instance of your app by setting an endpoint to scrap",
					Free:        &t,
					Bindable:    &t,
					Metadata: &domain.ServicePlanMetadata{
						DisplayName: "fetch-app",
					},
				},
			},
			Requires: []domain.RequiredPermission{},
			Metadata: &domain.ServiceMetadata{
				DisplayName:         "promfetcher",
				LongDescription:     "Fetch your prometheus metrics on each instance of your app.",
				DocumentationUrl:    b.baseURL + "/doc",
				SupportUrl:          "",
				ImageUrl:            "",
				ProviderDisplayName: "Orange",
			},
			DashboardClient: nil,
		},
	}, nil
}

func (b Broker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (domain.ProvisionedServiceSpec, error) {
	return domain.ProvisionedServiceSpec{}, nil
}

func (b Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	return domain.DeprovisionServiceSpec{}, nil
}

func (b Broker) GetInstance(ctx context.Context, instanceID string) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, nil
}

func (b Broker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	return domain.UpdateServiceSpec{}, nil
}

func (b Broker) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, nil
}

func (b Broker) Bind(ctx context.Context, instanceID, bindingID string, details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {
	if b.db == nil {
		return domain.Binding{}, fmt.Errorf("no db set broker unusable")
	}
	var params BrokerParams
	err := json.Unmarshal(details.RawParameters, &params)
	if err != nil && len(details.RawParameters) > 0 {
		return domain.Binding{}, fmt.Errorf("error when loading params: %s", err.Error())
	}

	if params.Endpoint != "" && params.Endpoint[0] != '/' {
		return domain.Binding{}, fmt.Errorf("endpoint must be a path starting with /")
	}

	b.db.Delete(models.AppEndpoint{}, "app_guid = ?", details.AppGUID)
	if params.Endpoint == "" {
		return domain.Binding{}, nil
	}

	err = b.db.Create(&models.AppEndpoint{
		GUID:     bindingID,
		AppGUID:  details.AppGUID,
		Endpoint: params.Endpoint,
	}).Error
	if err != nil {
		return domain.Binding{}, fmt.Errorf("error when getting creating app entry in db: %s", err.Error())
	}
	return domain.Binding{}, nil
}

func (b Broker) Unbind(ctx context.Context, instanceID, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {
	if b.db == nil {
		return domain.UnbindSpec{}, nil
	}
	b.db.Delete(models.AppEndpoint{}, "guid = ?", bindingID)
	return domain.UnbindSpec{}, nil
}

func (b Broker) GetBinding(ctx context.Context, instanceID, bindingID string) (domain.GetBindingSpec, error) {
	var appEndpoint models.AppEndpoint
	if b.db == nil {
		return domain.GetBindingSpec{}, fmt.Errorf("no db set broker unusable")
	}
	err := b.db.First(&appEndpoint, "guid = ?", bindingID).Error
	if err != nil {
		return domain.GetBindingSpec{}, fmt.Errorf("error when getting app in db: %s", err.Error())
	}
	if appEndpoint.GUID == "" {
		return domain.GetBindingSpec{
			Credentials: map[string]string{},
		}, nil
	}
	return domain.GetBindingSpec{
		Credentials: map[string]string{
			"endpoint": appEndpoint.Endpoint,
		},
	}, nil
}

func (b Broker) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, nil
}
