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
	InventoryID  string
	SystemID     string
	CreatedAtLT  string
	CreatedAtLTE string
	CreatedAtGT  string
	CreatedAtGTE string
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
	Account     string    `json:"account"`
	RequestID   string    `json:"request_id"`
	InventoryID string    `json:"inventory_id,omitempty"`
	SystemID    string    `json:"system_id,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	Status      string    `json:"status,omitempty"`
	StatusMsg   string    `json:"status_msg,omitempty"`
	Date        time.Time `json:"date,omitempty"`
}
