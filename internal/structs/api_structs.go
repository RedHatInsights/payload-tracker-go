package structs

import (
	"time"

	"github.com/redhatinsights/payload-tracker-go/internal/models"
)

// Query is a struct for holding query params
type Query struct {
	Page         int
	PageSize     int
	RequestID    string
	SortBy       string
	SortDir      string
	Account      string
	OrgID        string
	InventoryID  string
	SystemID     string
	CreatedAtLT  string
	CreatedAtLTE string
	CreatedAtGT  string
	CreatedAtGTE string

	Service   string
	Source    string
	Status    string
	StatusMsg string
	DateLT    string
	DateLTE   string
	DateGT    string
	DateGTE   string
}

// PayloadsData is the response for the /payloads endpoint
type PayloadsData struct {
	Count   int64             `json:"count"`
	Elapsed float64           `json:"elapsed"`
	Data    []models.Payloads `json:"data"`
}

// PayloadRetrievebyID is the response for the /payloads/{request_id} endpoint
type PayloadRetrievebyID struct {
	Data      []SinglePayloadData `json:"data"`
	Durations map[string]string   `json:"duration"`
}

type PayloadArchiveLink struct {
	Url string `json:"url"`
}

type PayloadKibanaLink struct {
	Url string `json:"url"`
}

type ArchiveLinkRole struct {
	Allowed bool `json:"allowed"`
}

type StatusesData struct {
	Count   int64            `json:"count"`
	Elapsed float64          `json:"elapsed"`
	Data    []StatusRetrieve `json:"data"`
}

// Error response struct for endpoints
type ErrorResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// SinglePayloadData is the data for a single payload
type SinglePayloadData struct {
	ID          uint      `json:"id,omitempty"`
	Service     string    `json:"service,omitempty"`
	Source      string    `json:"source,omitempty"`
	Account     string    `json:"account,omitempty"`
	OrgID       string    `json:"org_id,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
	InventoryID string    `json:"inventory_id,omitempty"`
	SystemID    string    `json:"system_id,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	Status      string    `json:"status,omitempty"`
	StatusMsg   string    `json:"status_msg,omitempty"`
	Date        time.Time `json:"date,omitempty"`
}

// StatusRetrieve returns a response for /payloads/statuses
type StatusRetrieve struct {
	RequestID string `json:"request_id,omitempty"`
	Status    string `json:"status,omitempty"`
	ID        string `json:"id,omitempty"`
	Service   string `json:"service,omitempty"`
	Source    string `json:"source,omitempty"`
	StatusMsg string `json:"status_msg,omitempty"`
	Date      string `json:"date,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
