package structs

import (
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

type PayloadsData struct {
	Count   int64             `json:"count"`
	Elapsed float64           `json:"elapsed"`
	Data    []models.Payloads `json:"data"`
}

type ErrorResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}
