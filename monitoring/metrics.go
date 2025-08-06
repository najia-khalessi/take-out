package monitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DBĎuraction = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "db_call_duration_seconds",
		Help: "Duration of database calls.",
	}, []string{"operation"})

	RedisDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "redis_call_duration_seconds",
		Help: "Duration of Redis calls.",
	}, []string{"operation"})

	LogQueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "log_queue_size",
		Help: "Current size of the log queue.",
	})

	LogsDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "logs_dropped_total",
		Help: "Total number of logs dropped due to a full queue.",
	})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"path", "method", "code"})
)

func RecordDBTime(operation string, f func() error) error {
	start := time.Now()
	err := f()
	duration := time.Since(start)
	DBĎuraction.WithLabelValues(operation).Observe(duration.Seconds())
	return err
}

func RecordRedisTime(operation string, f func() error) error {
	start := time.Now()
	err := f()
	duration := time.Since(start)
	RedisDuration.WithLabelValues(operation).Observe(duration.Seconds())
	return err
}

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		d := &responseData{
			status: 200,
		}
		lrw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   d,
		}
		next.ServeHTTP(&lrw, r)
		duration := time.Since(start)
		httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(d.status)).Inc()
		httpDuration.WithLabelValues(r.URL.Path).Observe(duration.Seconds())
	})
}

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
