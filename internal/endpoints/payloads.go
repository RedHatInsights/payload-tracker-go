package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/logging"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/queries"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	RetrievePayloads          = queries.RetrievePayloads
	RetrieveRequestIdPayloads = queries.RetrieveRequestIdPayloads
	Db                        = getDb
)

func CreatePayloadArchiveLinkHandler(cfg config.TrackerConfig) http.HandlerFunc {
	switch cfg.RequestConfig.RequestorImpl {
	case "storage-broker":
		return PayloadArchiveLink(RequestArchiveLink(cfg.StorageBrokerURL, cfg.StorageBrokerRequestTimeout))
	case "mock":
		return MockArchiveLink
	default:
		l.Log.Errorf("Requestor implementation %s not supported", cfg.RequestConfig.RequestorImpl)
		return nil
	}
}

// Payloads returns responses for the /payloads endpoint
func Payloads(w http.ResponseWriter, r *http.Request) {

	// init query with defaults and passed params
	start := time.Now()

	sortBy := r.URL.Query().Get("sort_by")
	incRequests()

	q, err := initQuery(r)

	if err != nil {
		writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%v", err), http.StatusBadRequest))
		return
	}

	// there is a different default for sortby when searching for payloads
	if sortBy == "" {
		q.SortBy = "created_at"
	}

	if !stringInSlice(q.SortBy, validAllSortBy) {
		message := "sort_by must be one of " + strings.Join(validAllSortBy, ", ")
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}
	if !stringInSlice(q.SortDir, validSortDir) {
		message := "sort_dir must be one of " + strings.Join(validSortDir, ", ")
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}

	if !validTimestamps(q, false) {
		message := "invalid timestamp format provided"
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}

	count, payloads := RetrievePayloads(Db(), q.Page, q.PageSize, q)
	duration := time.Since(start).Seconds()
	observeDBTime(time.Since(start))

	payloadsData := structs.PayloadsData{count, duration, payloads}

	dataJson, err := json.Marshal(payloadsData)
	if err != nil {
		l.Log.Error(err)
		writeResponse(w, http.StatusInternalServerError, getErrorBody("Internal Server Issue", http.StatusInternalServerError))
		return
	}

	writeResponse(w, http.StatusOK, string(dataJson))
}

// RequestIdPayloads returns a response for /payloads/{request_id}
func RequestIdPayloads(w http.ResponseWriter, r *http.Request) {

	reqID := chi.URLParam(r, "request_id")
	verbosity := r.URL.Query().Get("verbosity")

	q, err := initQuery(r)

	if err != nil {
		writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%v", err), http.StatusBadRequest))
		return
	}

	if !stringInSlice(q.SortBy, validIDSortBy) {
		message := "sort_by must be one of " + strings.Join(validIDSortBy, ", ")
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}
	if !stringInSlice(q.SortDir, validSortDir) {
		message := "sort_dir must be one of " + strings.Join(validSortDir, ", ")
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}

	payloads := RetrieveRequestIdPayloads(Db(), reqID, q.SortBy, q.SortDir, verbosity)

	if payloads == nil || len(payloads) == 0 {
		writeResponse(w, http.StatusNotFound, getErrorBody("payload with id: "+reqID+" not found", http.StatusNotFound))
		return
	}

	durations := queries.CalculateDurations(payloads)

	payloadsData := structs.PayloadRetrievebyID{Data: payloads, Durations: durations}

	dataJson, err := json.Marshal(payloadsData)
	if err != nil {
		l.Log.Error(err)
		writeResponse(w, http.StatusInternalServerError, getErrorBody("Internal Server Issue", http.StatusInternalServerError))
		return
	}

	writeResponse(w, http.StatusOK, string(dataJson))
}

// PayloadArchiveLink returns a response for /payloads/{request_id}/archiveLink
func PayloadArchiveLink(requestArchiveLink func(context.Context, string) (*structs.PayloadArchiveLink, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		reqID := chi.URLParam(r, "request_id")

		statusCode, err := checkForRole(r, config.Get().StorageBrokerURLRole)
		if err != nil {
			writeResponse(w, statusCode, getErrorBody(fmt.Sprintf("%v", err), statusCode))
			return
		}

		if !isValidUUID(reqID) {
			IncInvalidAPIRequestIDs()
			writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%s is not a valid UUID", reqID), http.StatusBadRequest))
			return
		}

		payloadArchiveLink, err := requestArchiveLink(r.Context(), reqID)
		if err != nil {
			l.Log.Errorf("Error getting archive link from storage-broker for request id: %s, error: %v", reqID, err)
			writeResponse(w, http.StatusInternalServerError, getErrorBody(fmt.Sprintf("%v", err), http.StatusInternalServerError))
			return
		}

		if payloadArchiveLink.Url == "" {
			writeResponse(w, http.StatusNotFound, getErrorBody("Payload not found", http.StatusNotFound))
			return
		}

		dataJson, err := json.Marshal(payloadArchiveLink)
		if err != nil {
			l.Log.Error(err)
			writeResponse(w, http.StatusInternalServerError, getErrorBody("Error converting parsed response to json", http.StatusInternalServerError))
			return
		}

		l.Log.Infof("Archive link generated for payload %s (identity %s)", reqID, hashIdentity(r.Header.Get("x-rh-identity")))
		writeResponse(w, http.StatusOK, string(dataJson))
	}
}

func MockArchiveLink(w http.ResponseWriter, r *http.Request) {
	reqID := chi.URLParam(r, "request_id")
	url := fmt.Sprintf("http://%s:%s/app/payload-tracker/api/v1/archive/%s", config.Get().Hostname, config.Get().PublicPort, reqID)

	response := &structs.PayloadArchiveLink{
		Url: url,
	}
	dataJson, _ := json.Marshal(response)

	writeResponse(w, http.StatusOK, string(dataJson))
}

func PayloadKibanaLink(w http.ResponseWriter, r *http.Request) {

	reqID := chi.URLParam(r, "request_id")

	if !isValidUUID(reqID) {
		IncInvalidAPIRequestIDs()
		writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%s is not a valid UUID", reqID), http.StatusBadRequest))
		return
	}

	service := r.URL.Query().Get("service")

	cfg := config.Get()
	serviceField := cfg.KibanaConfig.ServiceField
	kibanaUrl := cfg.KibanaConfig.DashboardURL
	kibanaIndex := cfg.KibanaConfig.Index

	kibanaQuery := "request_id:" + reqID
	if service != "" {
		kibanaQuery += fmt.Sprintf(" AND %s:%s", serviceField, service)
	}

	kibanaLink := fmt.Sprintf("%s?_g=(filters:!(),refreshInterval:(pause:!t,value:0),time:(from:now-24h,to:now))&_a=(columns:!(_source),filters:!(),index:'%s',interval:auto,query:(language:lucene,query:'%s'),sort:!('@timestamp',desc))", kibanaUrl, kibanaIndex, kibanaQuery)
	logging.Log.Debugf("Generated kibana link: %s", kibanaLink)

	payloadKibanaLink := structs.PayloadKibanaLink{
		Url: kibanaLink,
	}

	dataJson, err := json.Marshal(payloadKibanaLink)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, getErrorBody(fmt.Sprintf("%v", err), http.StatusInternalServerError))
		return
	}

	writeResponse(w, http.StatusOK, string(dataJson))
}
