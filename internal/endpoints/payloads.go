package endpoints

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/redhatinsights/payload-tracker-go/internal/db_methods"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
)

var (
	validSortBy    = []string{"created_at", "account", "system_id", "inventory_id", "service", "source", "status_msg", "date"}
	validAllSortBy = []string{"account", "inventory_id", "system_id", "created_at"}
	validIDSortBy  = []string{"service", "source", "status_msg", "date", "created_at"}
	validSortDir   = []string{"asc", "desc"}
)

// Query is a struct for holding query params
type Query struct {
	Page         int
	PageSize     int
	RequestID    string
	SortBy       string
	SortDir      string
	Account      string
	InventoryID  string
	SystemID     string
	CreatedAtLT  string
	CreatedAtLTE string
	CreatedAtGT  string
	CreatedAtGTE string
}

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

type PayloadsData struct {
	Count   int               `json:"count"`
	Elapsed float64           `json:"elapsed"`
	Data    []models.Payloads `json:"data"`
}

// initQuery intializes the query with default values
func initQuery(r *http.Request) Query {

	q := Query{
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

// Payloads returns responses for the /payloads endpoint
func Payloads(w http.ResponseWriter, r *http.Request) {

	// init query with defaults and passed params
	q := initQuery(r)
	sortBy := r.URL.Query().Get("sort_by")

	if q.SortBy != sortBy && stringInSlice(sortBy, validAllSortBy) {
		q.SortBy = sortBy
	}

	// TODO: do some database stuff
	payloads := db_methods.RetrievePayloads(q.Page, q.PageSize)

	payloadsData := PayloadsData{len(payloads), 0.763294877, payloads}

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
