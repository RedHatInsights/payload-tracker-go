package endpoints

import (
	"net/http"
	"strconv"
	"time"

	p "github.com/prometheus/client_golang/prometheus"
	pa "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requests = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_requests",
		Help: "Number of requests to the payload tracker.",
	}, []string{})

	dbElapsed = pa.NewHistogramVec(p.HistogramOpts{
		Name: "payload_tracker_db_seconds",
		Help: "Number of seconds spent waiting on a db response",
	}, []string{})

	responseCodes = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_responses",
		Help: "Count of response codes by code",
	}, []string{"code"})
)

type metricTrackingResponseWriter struct {
	Wrapped   http.ResponseWriter
	UserAgent string
}

func incRequests() {
	requests.With(p.Labels{}).Inc()
}

func observeDBTime(elapsed time.Duration) {
	dbElapsed.With(p.Labels{}).Observe(elapsed.Seconds())
}

func (m *metricTrackingResponseWriter) Header() http.Header {
	return m.Wrapped.Header()
}

func (m *metricTrackingResponseWriter) WriteHeader(statusCode int) {
	responseCodes.With(p.Labels{"code": strconv.Itoa(statusCode)}).Inc()
	m.Wrapped.WriteHeader(statusCode)
}

func (m *metricTrackingResponseWriter) Write(b []byte) (int, error) {
	return m.Wrapped.Write(b)
}

// ResponseMetricsMiddleware wraps the ResponseWriter such that metrics for each
// response type get tracked
func ResponseMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &metricTrackingResponseWriter{
 			Wrapped:   w,
		}
		next.ServeHTTP(ww, r)
	})
}