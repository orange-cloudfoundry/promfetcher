package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

const ProcessWeb = "web"

var mu = sync.RWMutex{}

type Routes map[Uri][]*Route

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
	Host                string     `json:"host"`
}

func (rts Routes) FindByOrgSpaceName(org, space, name string) []*Route {
	mu.RLock()
	defer mu.RUnlock()

	finalRoutes := make([]*Route, 0)
	exist := make(map[string]bool)
	for u, routes := range rts {
		for _, route := range routes {
			if route == nil {
				log.Debugf("Route is nil for %s", string(u))
				continue
			}
			route.URL = string(u)
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

func (rts Routes) FindById(appId string) []*Route {
	mu.RLock()
	defer mu.RUnlock()

	finalRoutes := make([]*Route, 0)
	exist := make(map[string]bool)
	for u, routes := range rts {
		for _, route := range routes {
			if route == nil {
				log.Debugf("Route is nil for %s", string(u))
				continue
			}
			route.URL = string(u)
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

func (rts Routes) FindByRouteName(routeName string) []*Route {
	mu.RLock()
	defer mu.RUnlock()

	routeKey := Uri(routeName).RouteKey()
	finalRoutes, ok := rts[routeKey]
	if !ok {
		return []*Route{}
	}
	return finalRoutes
}

func (rts Routes) Find(appIdOrPathOrName string) []*Route {
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

func (rts Routes) RegisterRoute(uri Uri, route *Route) {
	mu.Lock()
	defer mu.Unlock()

	if route == nil {
		log.Warn("Cannot register nil route")
		return
	}
	routekey := uri.RouteKey()
	routes, ok := rts[routekey]

	if ok {
		found := false
		for idx, r := range routes {
			if route.Equal(r) {
				found = true
				if route.NeedUpdate(r) {
					// route is updated
					log.Debugf("update route for uri %s and instance %s", string(uri), route.Tags.InstanceID)
					routes[idx] = route
					rts[routekey] = routes
					break
				}
			}
		}
		if !found {
			routes = append(routes, route)
			log.Debugf("register route for uri %s and instance %s", string(uri), route.Tags.InstanceID)
			rts[routekey] = routes
		}
	} else {
		r := make([]*Route, 0)
		r = append(r, route)
		log.Debugf("register route for uri %s and instance %s", string(uri), route.Tags.InstanceID)
		rts[routekey] = r
	}
}

func (rts Routes) UnregisterRoute(uri Uri, route *Route) {
	mu.Lock()
	defer mu.Unlock()

	if route == nil {
		log.Warn("Cannot unregister nil route")
		return
	}
	routekey := uri.RouteKey()
	routes, ok := rts[routekey]

	if ok {
		for idx, r := range routes {
			if route.Equal(r) {
				log.Debugf("unregister route for uri %s and instance %s", string(uri), route.Tags.InstanceID)
				// Trick for deleting an element from a slice
				size := len(routes)
				routes[idx] = routes[size-1]
				routes[size-1] = nil
				routes = routes[:size-1]
				rts[uri] = routes
				break
			}
		}
	} else {
		log.Infof("no route to unregister (%s)", uri)
	}
}

func (rts Routes) String() string {
	mu.RLock()
	defer mu.RUnlock()

	finalStr := "{"
	for u, routes := range rts {
		finalStr += fmt.Sprintf("\"%s\": [", u)
		for _, route := range routes {
			jsonRoute, jsonErr := json.Marshal(route)
			if jsonErr != nil {
				return "Error to generate Json"
			}
			finalStr += fmt.Sprintf("%s,", jsonRoute)
		}
		finalStr = strings.TrimRight(finalStr, ",") + "],"
	}
	finalStr = strings.TrimRight(finalStr, ",") + "}"
	return finalStr
}

func (r *Route) Equal(r2 *Route) bool {
	if r2 == nil {
		return false
	}

	return r.PrivateInstanceID == r2.PrivateInstanceID &&
		r.ServerCertDomainSan == r2.ServerCertDomainSan &&
		r.Address == r2.Address &&
		r.Host == r2.Host &&
		r.Tags.InstanceID == r2.Tags.InstanceID &&
		r.Tags.ProcessInstanceID == r2.Tags.ProcessInstanceID
}

func (r *Route) NeedUpdate(r2 *Route) bool {
	if r2 == nil {
		return false
	}

	return r.Tags.AppID != r2.Tags.AppID ||
		r.Tags.AppName != r2.Tags.AppName ||
		r.Tags.OrganizationID != r2.Tags.OrganizationID ||
		r.Tags.OrganizationName != r2.Tags.OrganizationName ||
		r.Tags.SpaceID != r2.Tags.SpaceID ||
		r.Tags.SpaceName != r2.Tags.SpaceName
}
