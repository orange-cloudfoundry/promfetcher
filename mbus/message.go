package mbus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/orange-cloudfoundry/promfetcher/models"
)

type Message struct {
	App                     string       `json:"app"`
	AvailabilityZone        string       `json:"availability_zone"`
	EndpointUpdatedAtNs     int64        `json:"endpoint_updated_at_ns"`
	Host                    string       `json:"host"`
	IsolationSegment        string       `json:"isolation_segment"`
	Port                    uint16       `json:"port"`
	PrivateInstanceID       string       `json:"private_instance_id"`
	PrivateInstanceIndex    string       `json:"private_instance_index"`
	Protocol                string       `json:"protocol"`
	RouteServiceURL         string       `json:"route_service_url"`
	ServerCertDomainSAN     string       `json:"server_cert_domain_san"`
	StaleThresholdInSeconds int          `json:"stale_threshold_in_seconds"`
	TLSPort                 uint16       `json:"tls_port"`
	Tags                    models.Tags  `json:"tags"`
	Uris                    []models.Uri `json:"uris"`
}

func (m *Message) MakeRoute(http2Enabled bool) (*models.Route, error) {
	port, useTLS, err := m.port()
	if err != nil {
		return nil, err
	}

	return &models.Route{
		PrivateInstanceID:   m.PrivateInstanceID,
		Tags:                m.Tags,
		ServerCertDomainSan: m.ServerCertDomainSAN,
		Address:             fmt.Sprintf("%s:%d", m.Host, port),
		TLS:                 useTLS,
		TTL:                 m.StaleThresholdInSeconds,
		Host:                m.Host,
	}, nil
}

// ValidateMessage checks to ensure the message is valid
func (m *Message) ValidateMessage() bool {
	return m.RouteServiceURL == "" || strings.HasPrefix(m.RouteServiceURL, "https")
}

// Prefer TLS Port instead of HTTP Port in Message
func (m *Message) port() (uint16, bool, error) {
	if m.TLSPort != 0 {
		return m.TLSPort, true, nil
	}
	return m.Port, false, nil
}

func CreateMessage(data []byte) (*Message, error) {
	var msg Message
	jsonErr := json.Unmarshal(data, &msg)
	if jsonErr != nil {
		return nil, jsonErr
	}

	if !msg.ValidateMessage() {
		return nil, errors.New("Unable to validate message. route_service_url must be https")
	}

	return &msg, nil
}
