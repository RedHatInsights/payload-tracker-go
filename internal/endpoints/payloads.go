package endpoints

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redhatinsights/payload-tracker-go/internal/db_methods"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	validSortBy    = []string{"created_at", "account", "system_id", "inventory_id", "service", "source", "status_msg", "date"}
	validAllSortBy = []string{"account", "inventory_id", "system_id", "created_at"}
	validIDSortBy  = []string{"service", "source", "status_msg", "date", "created_at"}
	validSortDir   = []string{"asc", "desc"}
)

var (
	RetrievePayloads          = db_methods.RetrievePayloads
	RetrieveRequestIdPayloads = db_methods.RetrieveRequestIdPayloads
)

// initQuery intializes the query with default values
func initQuery(r *http.Request) (structs.Query, error) {

	q := structs.Query{
		Page:         0,
		PageSize:     10,
		SortBy:       "created_at",
		SortDir:      "desc",
		InventoryID:  r.URL.Query().Get("inventory_id"),
		SystemID:     r.URL.Query().Get("system_id"),
		CreatedAtLT:  r.URL.Query().Get("created_at_lt"),
		CreatedAtGT:  r.URL.Query().Get("created_at_gt"),
		CreatedAtLTE: r.URL.Query().Get("created_at_lte"),
		CreatedAtGTE: r.URL.Query().Get("created_at_gte"),
		Account:      r.URL.Query().Get("account"),
	}

	var err error

	if r.URL.Query().Get("sort_by") != "" || stringInSlice(r.URL.Query().Get("sort_by"), validSortBy) {
		q.SortBy = r.URL.Query().Get("sort_by")
	}

	if r.URL.Query().Get("sort_dir") != "" || stringInSlice(r.URL.Query().Get("sort_dir"), validSortDir) {
		q.SortDir = r.URL.Query().Get("sort_dir")
	}

	if r.URL.Query().Get("page") != "" {
		q.Page, err = strconv.Atoi(r.URL.Query().Get("page"))
	}

	if r.URL.Query().Get("page_size") != "" {
		q.PageSize, err = strconv.Atoi(r.URL.Query().Get("page_size"))
	}

	return q, err
}

func getErrorBody(message string, status int) string {
	errBody := structs.ErrorResponse{
		Title:   http.StatusText(status),
		Message: message,
		Status:  status,
	}

	errBodyJson, _ := json.Marshal(errBody)
	return string(errBodyJson)
}

// Check for value in a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Check timestamp format
func validTimestamps(q structs.Query) bool {
	timestampQueries := []string{q.CreatedAtLT, q.CreatedAtGT, q.CreatedAtLTE, q.CreatedAtGTE}

	for _, ts := range timestampQueries {
		if ts != "" {
			_, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return false
			}
		}
	}
	return true
}

// Write HTTP Response
func writeResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(message))
}

// Payloads returns responses for the /payloads endpoint
func Payloads(w http.ResponseWriter, r *http.Request) {

	// init query with defaults and passed params
	start := time.Now()

	q, err := initQuery(r)

	if err != nil {
		writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%v", err), http.StatusBadRequest))
		return
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

	if !validTimestamps(q) {
		message := "invalid timestamp format provided"
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}

	// TODO: do some database stuff
	count, payloads := RetrievePayloads(q.Page, q.PageSize, q)
	duration := time.Since(start).Seconds()

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
	sortBy := r.URL.Query().Get("sort_by")

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

	// there is a different default for sortby when searching for single payloads
	if sortBy == "" {
		q.SortBy = "date"
	}

	payloads := RetrieveRequestIdPayloads(reqID, q.SortBy, q.SortDir)
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
