package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/redhatinsights/payload-tracker-go/internal/db_methods"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	RetrievePayloads          = db_methods.RetrievePayloads
	RetrieveRequestIdPayloads = db_methods.RetrieveRequestIdPayloads
)

var (
	verbosity string = "0"
)

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

	count, payloads := RetrievePayloads(q.Page, q.PageSize, q)
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

// SinglePayload returns a resposne for /payloads/{request_id}
func RequestIdPayloads(w http.ResponseWriter, r *http.Request) {

	reqID := chi.URLParam(r, "request_id")
	verbosity = r.URL.Query().Get("verbosity")

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

	payloads := RetrieveRequestIdPayloads(reqID, q.SortBy, q.SortDir, verbosity)
	durations := db_methods.CalculateDurations(payloads)

	payloadsData := structs.PayloadRetrievebyID{Data: payloads, Durations: durations}

	dataJson, err := json.Marshal(payloadsData)
	if err != nil {
		l.Log.Error(err)
		writeResponse(w, http.StatusInternalServerError, getErrorBody("Internal Server Issue", http.StatusInternalServerError))
		return
	}

	writeResponse(w, http.StatusOK, string(dataJson))
}
