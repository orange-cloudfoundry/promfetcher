// Code generated by counterfeiter. DO NOT EDIT.
package fetchersfakes

import (
	"sync"

	"github.com/orange-cloudfoundry/promfetcher/fetchers"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

type FakeRoutesFetch struct {
	RoutesStub        func() models.Routes
	routesMutex       sync.RWMutex
	routesArgsForCall []struct {
	}
	routesReturns struct {
		result1 models.Routes
	}
	routesReturnsOnCall map[int]struct {
		result1 models.Routes
	}
	RunStub        func()
	runMutex       sync.RWMutex
	runArgsForCall []struct {
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRoutesFetch) Routes() models.Routes {
	fake.routesMutex.Lock()
	ret, specificReturn := fake.routesReturnsOnCall[len(fake.routesArgsForCall)]
	fake.routesArgsForCall = append(fake.routesArgsForCall, struct {
	}{})
	fake.recordInvocation("Routes", []interface{}{})
	fake.routesMutex.Unlock()
	if fake.RoutesStub != nil {
		return fake.RoutesStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.routesReturns
	return fakeReturns.result1
}

func (fake *FakeRoutesFetch) RoutesCallCount() int {
	fake.routesMutex.RLock()
	defer fake.routesMutex.RUnlock()
	return len(fake.routesArgsForCall)
}

func (fake *FakeRoutesFetch) RoutesCalls(stub func() models.Routes) {
	fake.routesMutex.Lock()
	defer fake.routesMutex.Unlock()
	fake.RoutesStub = stub
}

func (fake *FakeRoutesFetch) RoutesReturns(result1 models.Routes) {
	fake.routesMutex.Lock()
	defer fake.routesMutex.Unlock()
	fake.RoutesStub = nil
	fake.routesReturns = struct {
		result1 models.Routes
	}{result1}
}

func (fake *FakeRoutesFetch) RoutesReturnsOnCall(i int, result1 models.Routes) {
	fake.routesMutex.Lock()
	defer fake.routesMutex.Unlock()
	fake.RoutesStub = nil
	if fake.routesReturnsOnCall == nil {
		fake.routesReturnsOnCall = make(map[int]struct {
			result1 models.Routes
		})
	}
	fake.routesReturnsOnCall[i] = struct {
		result1 models.Routes
	}{result1}
}

func (fake *FakeRoutesFetch) Run() {
	fake.runMutex.Lock()
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
	}{})
	fake.recordInvocation("Run", []interface{}{})
	fake.runMutex.Unlock()
	if fake.RunStub != nil {
		fake.RunStub()
	}
}

func (fake *FakeRoutesFetch) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeRoutesFetch) RunCalls(stub func()) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = stub
}

func (fake *FakeRoutesFetch) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.routesMutex.RLock()
	defer fake.routesMutex.RUnlock()
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeRoutesFetch) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ fetchers.RoutesFetch = new(FakeRoutesFetch)