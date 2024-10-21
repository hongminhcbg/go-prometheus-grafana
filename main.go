package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	prometheus.Register(responseStatus)
	prometheus.Register(requestDuration)
}

var (
	_              http.Handler = (*userService)(nil)
	rawBodySuccess              = []byte(`{"data": null}`)
)

var responseStatus = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "response_status",
	Help: "count response status from server",
}, []string{"xxx"})

var mockStatus = map[int]int{
	0: http.StatusConflict,
	1: http.StatusBadRequest,
	2: http.StatusBadGateway,
	3: http.StatusOK,
	4: http.StatusAccepted,
	5: http.StatusInternalServerError,
	6: http.StatusForbidden,
}

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "request_duration",
		Help:    "counter request duration",
		Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 5, 10},
	},
	[]string{"request_path"},
)

type userService struct{}

func middlewaresLoggingRequestDuration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		timer := prometheus.NewTimer(requestDuration.WithLabelValues(r.RequestURI))
		defer timer.ObserveDuration()

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func (s userService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	x := rand.Int() % 7
	time.Sleep(1000 * time.Duration(x) * time.Millisecond)
	statusResp := mockStatus[x]
	responseStatus.WithLabelValues(fmt.Sprintf("%d", statusResp)).Inc()
	w.WriteHeader(statusResp)
	_, _ = w.Write(rawBodySuccess)
}

func fakeClient() {
	t := time.NewTicker(5 * time.Second).C
	for range t {
		http.Get("http://0.0.0.0:8080/users")
	}
}

func main() {
	s := &userService{}
	router := mux.NewRouter()
	// Prometheus endpoint
	router.Use(middlewaresLoggingRequestDuration)
	router.Path("/metrics").Handler(promhttp.Handler())
	router.Path("/users").Handler(s)

	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-osSig:
			{
				log.Println("received signal kill")
				os.Exit(0)
			}
		}
	}()

	go fakeClient()

	fmt.Println("Serving requests on port 8080")
	err := http.ListenAndServe(":8080", router)
	log.Fatal(err)
}
