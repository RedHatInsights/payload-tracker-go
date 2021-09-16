package db_methods

import (
	"fmt"
	// "encoding/json"
	"strings"
	// "time"

	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)


var (
	payloadsFields = []string{"payloads.id", "payloads.request_id", "payloads.account", "payloads.request_id", "payloads.system_id"}
	payloadStatusesFields = []string{"payload_statuses.status_msg", "payload_statuses.date", "payload_statuses.created_at"}
	otherFields = []string{"services.name as service", "sources.name as source", "statuses.name as status"}
)

func RetrievePayloads(page int, pageSize int, apiQuery structs.Query) (int64, []models.Payloads) {
	var count int64
	var payloads []models.Payloads

	dbQuery := db.DB

	// query chaining
	if apiQuery.Account != "" {
		dbQuery = dbQuery.Where("account = ?", apiQuery.Account)
	}
	if apiQuery.InventoryID != "" {
		dbQuery = dbQuery.Where("inventory_id = ?", apiQuery.InventoryID)
	}
	if apiQuery.SystemID != "" {
		dbQuery = dbQuery.Where("system_id = ?", apiQuery.SystemID)
	}

	if apiQuery.CreatedAtLT != "" {
		dbQuery = dbQuery.Where("created_at < ?", apiQuery.CreatedAtLT)
	}
	if apiQuery.CreatedAtLTE != "" {
		dbQuery = dbQuery.Where("created_at <= ?", apiQuery.CreatedAtLTE)
	}
	if apiQuery.CreatedAtGT != "" {
		dbQuery = dbQuery.Where("created_at > ?", apiQuery.CreatedAtGT)
	}
	if apiQuery.CreatedAtGTE != "" {
		dbQuery = dbQuery.Where("created_at >= ?", apiQuery.CreatedAtGTE)
	}

	orderString := fmt.Sprintf("%s %s", apiQuery.SortBy, apiQuery.SortDir)

	dbQuery.Order(orderString).Limit(pageSize).Offset(pageSize * page).Find(&payloads).Count(&count)

	return count, payloads
}

func RetrieveSinglePayload(reqID string, sortBy string, sortDir string) []structs.SinglePayloadData {
	var payloads []structs.SinglePayloadData

	dbQuery := db.DB

	fields := strings.Join(payloadsFields,",")+","+strings.Join(payloadStatusesFields,",")+","+strings.Join(otherFields,",")
	dbQuery = dbQuery.Table("payload_statuses").Select(fields).Joins("JOIN payloads on payload_statuses.payload_id = payloads.id")
	dbQuery = dbQuery.Joins("JOIN services on payload_statuses.service_id = services.id").Joins("JOIN sources on payload_statuses.source_id = sources.id").Joins("JOIN statuses on payload_statuses.status_id = statuses.id")

	orderString := fmt.Sprintf("%s %s", sortBy, sortDir)

	dbQuery.Where("payloads.request_id = ?", reqID).Order(orderString).Scan(&payloads)
	fmt.Println("Request_id", reqID, orderString)
	return payloads
}
