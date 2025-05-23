package main

import (
	"database/sql"
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
	"github.com/prometheus/client_golang/prometheus/push"

	_ "github.com/go-sql-driver/mysql"
)

// max by (db_stats) (db_stats{job="server"})

func init() {
	prometheus.Register(responseStatus)
	prometheus.Register(requestDuration)
	prometheus.Register(dbStats)
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

var dbStats = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "db_stats",
}, []string{"db_stats"})

type (
	userService   struct{}
	userServiceV2 struct {
		db *sql.DB
	}
)

func newUserSvcWithSql() *userServiceV2 {
	db, err := sql.Open("mysql", "root:12345678@tcp(localhost:3306)/users")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		panic(err)
	}

	log.Println(db.Stats())
	ans := &userServiceV2{
		db: db,
	}

	ans.metrics()
	return ans
}

func (s *userServiceV2) metrics() {
	t := time.NewTicker(2 * time.Second).C
	go func() {
		for {
			select {
			case <-t:
				stats := s.db.Stats()
				log.Println("db stats", stats)
				dbStats.WithLabelValues("Idle").Set(float64(stats.Idle))
				dbStats.WithLabelValues("inuse").Set(float64(stats.InUse))
				dbStats.WithLabelValues("OpenConnections").Set(float64(stats.OpenConnections))
				dbStats.WithLabelValues("WaitCount").Set(float64(stats.WaitCount))
			}
		}
	}()
}

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

func (s *userServiceV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	ro, err := s.db.Query("SELECT * FROM users")
	if err != nil {
		panic(err)
	}

	defer ro.Close()

	x := rand.Int() % 7
	time.Sleep(1000 * time.Duration(x) * time.Millisecond)
	statusResp := mockStatus[x]
	responseStatus.WithLabelValues(fmt.Sprintf("%d", statusResp)).Inc()
	w.WriteHeader(statusResp)
	_, _ = w.Write(rawBodySuccess)
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

func initPrometheusPush() error {
	pusher := push.New(os.Getenv("PUSH_HOST"), "pushgateway").Gatherer(prometheus.DefaultGatherer)
	for {
		time.Sleep(3 * time.Second)
		pusher.Add()
		log.Println("pushed metrics")
	}
}

func main() {
	// s := &userService{}
	s := newUserSvcWithSql()
	router := mux.NewRouter()
	// Prometheus endpoint
	router.Use(middlewaresLoggingRequestDuration)
	router.Path("/metrics").Handler(promhttp.Handler())
	router.Path("/users").Handler(s)
	log.Println("IS_PUSH_METRIC", os.Getenv("IS_PUSH_METRIC"))
	if os.Getenv("IS_PUSH_METRIC") == "true" {
		go initPrometheusPush()
	}

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
