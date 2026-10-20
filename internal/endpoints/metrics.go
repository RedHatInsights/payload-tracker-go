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

	apiInvalidRequestIDs = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_api_invalid_request_IDs",
		Help: "Number of invalid request IDs recieved by the payload tracker archive link endpoint.",
	}, []string{})

	consumerInvalidRequestIDs = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_consumer_invalid_request_IDs",
		Help: "Number of invalid request IDs recieved by the payload tracker consumer.",
	}, []string{})

	dbElapsed = pa.NewHistogramVec(p.HistogramOpts{
		Name: "payload_tracker_db_seconds",
		Help: "Number of seconds spent waiting on a db response",
	}, []string{})

	messagesProcessed = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_messages_processed",
		Help: "Count of total messages processed",
	}, []string{})

	messageProcessElapsed = pa.NewHistogramVec(p.HistogramOpts{
		Name: "payload_tracker_message_process_seconds",
		Help: "Number of seconds spent processing messages",
	}, []string{})

	messageProcessError = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_message_process_errors",
		Help: "Count of message process errors",
	}, []string{})

	responseCodes = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_responses",
		Help: "Count of response codes by code",
	}, []string{"code"})

	consumedMessages = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_consumed_messages",
		Help: "Number of messages consumed by payload tracker",
	}, []string{})

	consumeError = pa.NewCounterVec(p.CounterOpts{
		Name: "payload_tracker_consume_errors",
		Help: "Number of consumer errors encountered",
	}, []string{})
)

type metricTrackingResponseWriter struct {
	Wrapped   http.ResponseWriter
	UserAgent string
}

func incRequests() {
	requests.With(p.Labels{}).Inc()
}

// IncConsumedMessages increments the message count by 1
func IncConsumedMessages() {
	consumedMessages.With(p.Labels{}).Inc()
}

// IncConsumeFailure increments the failure count by 1
func IncConsumeErrors() {
	consumeError.With(p.Labels{}).Inc()
}

// IncMessageProcessed  increments the messages processed count by 1
func IncMessagesProcessed() {
	messagesProcessed.With(p.Labels{}).Inc()
}

// IncMessageProcessErrors increments the error count by 1
func IncMessageProcessErrors() {
	messageProcessError.With(p.Labels{}).Inc()
}

func IncInvalidConsumerRequestIDs() {
	consumerInvalidRequestIDs.With(p.Labels{}).Inc()
}

func IncInvalidAPIRequestIDs() {
	apiInvalidRequestIDs.With(p.Labels{}).Inc()
}

func observeDBTime(elapsed time.Duration) {
	dbElapsed.With(p.Labels{}).Observe(elapsed.Seconds())
}

func ObserveMessageProcessTime(elapsed time.Duration) {
	messageProcessElapsed.With(p.Labels{}).Observe(elapsed.Seconds())
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
			Wrapped: w,
		}
		next.ServeHTTP(ww, r)
	})
}
