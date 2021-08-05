package healthchecks

import (
	"net/http"
	"sync"
)

type Status uint64

const (
	Initializing Status = iota
	Healthy
	Degraded
)

type HealthCheck struct {
	mu     sync.RWMutex // to lock health r/w
	health Status
}

func NewHealthCheck() *HealthCheck {
	return &HealthCheck{
		mu:     sync.RWMutex{},
		health: Initializing,
	}
}

func (h *HealthCheck) Health() Status {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.health
}

func (h *HealthCheck) SetHealth(s Status) {
	h.mu.Lock()

	if h.health == Degraded {
		h.mu.Unlock()
		return
	}

	h.health = s
	h.mu.Unlock()

}

func (h *HealthCheck) String() string {
	switch h.Health() {
	case Initializing:
		return "Initializing"
	case Healthy:
		return "Healthy"
	case Degraded:
		return "Degraded"
	default:
		panic("health: unknown status")
	}
}

func (h *HealthCheck) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Cache-Control", "private, max-age=0")
	rw.Header().Set("Expires", "0")

	if h.Health() != Healthy {
		rw.WriteHeader(http.StatusServiceUnavailable)
		r.Close = true
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("ok\n"))
	r.Close = true
}
