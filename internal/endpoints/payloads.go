package endpoints

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"fmt"

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

// ReturnData is the response for the endpoint
type ReturnData struct {
	Count               int                   `json:"count"`
	Elapsed             string                `json:"elapsed"`
	PayloadRetrieve     []PayloadRetrieve     `json:"data"`
	PayloadRetrievebyID []PayloadRetrievebyID `json:"data"`
	StatusRetrieve      []StatusRetrieve      `json:"data"`
}

// PayloadRetrieve is the data for all payloads
type PayloadRetrieve struct {
	RequestID   string `json:"request_id"`
	Account     string `json:"account"`
	InventoryID string `json:"inventory_id,omitempty"`
	SystemID    string `json:"system_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// PayloadRetrievebyID is the data for a single payload
type PayloadRetrievebyID struct {
	ID          string `json:"id,omitempty"`
	Service     string `json:"service,omitempty"`
	Source      string `json:"source,omitempty"`
	Account     string `json:"account"`
	RequestID   string `json:"request_id"`
	InventoryID string `json:"inventory_id,omitempty"`
	SystemID    string `json:"system_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	Status      string `json:"status,omitempty"`
	StatusMsg   string `json:"status_msg,omitempty"`
	Date        string `json:"date,omitempty"`
}

// DurationsRetrieve hold the time spend in a given service
type DurationsRetrieve struct {
	Service   string `json:"service"`
	TimeDelta string `json:"timedelta"`
}

// initQuery intializes the query with default values
func initQuery(r *http.Request) structs.Query {

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

	if r.URL.Query().Get("sort_by") != "" || stringInSlice(r.URL.Query().Get("sort_by"), validSortBy) {
		q.SortBy = r.URL.Query().Get("sort_by")
	}

	if r.URL.Query().Get("sort_dir") != "" || stringInSlice(r.URL.Query().Get("sort_dir"), validSortDir) {
		q.SortDir = r.URL.Query().Get("sort_dir")
	}

	if r.URL.Query().Get("page") != "" {
		q.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	}

	if r.URL.Query().Get("page_size") != "" {
		q.PageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	}

	return q
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
		if ts != ""{
			_, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				fmt.Println(err)
				return false
			}
		}
	}
	return true
}

// Payloads returns responses for the /payloads endpoint
func Payloads(w http.ResponseWriter, r *http.Request) {

	// init query with defaults and passed params
	start := time.Now()

	q := initQuery(r)
	sortBy := r.URL.Query().Get("sort_by")

	if !stringInSlice(sortBy, validAllSortBy) {
		w.WriteHeader(http.StatusBadRequest)
		message := "sort_by must be one of "+strings.Join(validAllSortBy, ", ")
		w.Write([]byte(message))
		return
	}
	if !stringInSlice(q.SortDir, validSortDir) {
		w.WriteHeader(http.StatusBadRequest)
		message := "sort_dir must be one of "+strings.Join(validSortDir, ", ")
		w.Write([]byte(message))
		return
	}

	if !validTimestamps(q) {
		w.WriteHeader(http.StatusBadRequest)
		message := "invalid timestamp format provided"
		w.Write([]byte(message))
		return
	}


	if q.SortBy != sortBy {
		q.SortBy = sortBy
	}

	// TODO: do some database stuff
	payloads := db_methods.RetrievePayloads(q.Page, q.PageSize, q)
	duration := time.Since(start).Seconds()

	payloadsData := structs.PayloadsData{len(payloads), duration, payloads}

	dataJson, err := json.Marshal(payloadsData)
	if err != nil {
		l.Log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Issue"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(dataJson)))
}

// SinglePayload returns a resposne for /payloads/{request_id}
func SinglePayload(w http.ResponseWriter, r *http.Request) {

	reqID := r.URL.Query().Get("request_id")
	sortBy := r.URL.Query().Get("sort_by")

	q := initQuery(r)

	// there is a different default for sortby when searching for single payloads
	// we first check that the sortby param is valid, then set to either that value or the default
	if q.SortBy != sortBy && stringInSlice(sortBy, validIDSortBy) {
		q.SortBy = sortBy
	} else {
		q.SortBy = "date"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(reqID))
}
