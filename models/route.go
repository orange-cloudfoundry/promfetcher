package models

import (
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const ProcessWeb = "web"

type Routes map[string][]Route

type Tags struct {
	ProcessType       string `json:"process_type"`
	ProcessInstanceID string `json:"process_instance_id"`
	Component         string `json:"component"`
	InstanceID        string `json:"instance_id"`
	SpaceName         string `json:"space_name"`
	OrganizationID    string `json:"organization_id"`
	ProcessID         string `json:"process_id"`
	OrganizationName  string `json:"organization_name"`
	SourceID          string `json:"source_id"`
	AppID             string `json:"app_id"`
	AppName           string `json:"app_name"`
	SpaceID           string `json:"space_id"`
}

type Route struct {
	PrivateInstanceID   string     `json:"private_instance_id"`
	Tags                Tags       `json:"tags"`
	ServerCertDomainSan string     `json:"server_cert_domain_san"`
	Address             string     `json:"address"`
	TLS                 bool       `json:"tls"`
	TTL                 int        `json:"ttl"`
	URL                 string     `json:"-"`
	URLParams           url.Values `json:"-"`
	MetricsPath         string     `json:"-"`
}

func (rts Routes) FindByOrgSpaceName(org, space, name string) []Route {
	finalRoutes := make([]Route, 0)
	exist := make(map[string]bool)
	for u, routes := range rts {
		for _, route := range routes {
			route.URL = u
			if route.Tags.ProcessType != ProcessWeb {
				continue
			}
			if _, ok := exist[route.Address]; ok {
				continue
			}
			if route.Tags.OrganizationName != org ||
				route.Tags.SpaceName != space ||
				route.Tags.AppName != name {
				continue
			}
			exist[route.Address] = true
			finalRoutes = append(finalRoutes, route)
		}
	}
	return finalRoutes
}

func (rts Routes) FindById(appId string) []Route {
	finalRoutes := make([]Route, 0)
	exist := make(map[string]bool)
	for u, routes := range rts {
		for _, route := range routes {
			route.URL = u
			if route.Tags.ProcessType != ProcessWeb {
				continue
			}
			if _, ok := exist[route.Address]; ok {
				continue
			}
			if route.Tags.AppID != appId {
				continue
			}
			exist[route.Address] = true
			finalRoutes = append(finalRoutes, route)
		}
	}
	return finalRoutes
}

func (rts Routes) FindByRouteName(routeName string) []Route {
	finalRoutes, ok := rts[routeName]
	if !ok {
		return []Route{}
	}
	return finalRoutes
}

func (rts Routes) Find(appIdOrPathOrName string) []Route {
	tmpContent, err := url.PathUnescape(appIdOrPathOrName)
	if err == nil {
		appIdOrPathOrName = tmpContent
	}
	splitContent := strings.Split(appIdOrPathOrName, "/")
	if len(splitContent) == 3 {
		return rts.FindByOrgSpaceName(splitContent[0], splitContent[1], splitContent[2])
	}
	// if can be parsed as uuid that's a uuid
	_, err = uuid.Parse(appIdOrPathOrName)
	if err == nil {
		return rts.FindById(appIdOrPathOrName)
	}
	return rts.FindByRouteName(appIdOrPathOrName)
}
