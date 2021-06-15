package install

import (
	"net/http"
	"sync/atomic"
)

// StartServer initializes and starts a web server that exposes liveness and readiness endpoints at port 8000.
func StartServer() *atomic.Value {
	router := http.NewServeMux()
	isReady := initRouter(router)

	go func() {
		_ = http.ListenAndServe(":"+Port, router)
	}()

	return isReady
}

// Sets isReady to true.
func SetReady(isReady *atomic.Value) {
	isReady.Store(true)
}

// Sets isReady to false.
func SetNotReady(isReady *atomic.Value) {
	isReady.Store(false)
}

func initRouter(router *http.ServeMux) *atomic.Value {
	isReady := &atomic.Value{}
	isReady.Store(false)

	router.HandleFunc(LivenessEndpoint, healthz)
	router.HandleFunc(ReadinessEndpoint, readyz(isReady))

	return isReady
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readyz(isReady *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if isReady == nil || !isReady.Load().(bool) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
