package db_methods

import (
	"fmt"

	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

func RetrievePayloads(page int, pageSize int, apiQuery structs.Query) (int64, []models.Payloads) {
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

	var count int64

	orderString := fmt.Sprintf("%s %s", apiQuery.SortBy, apiQuery.SortDir)

	dbQuery.Order(orderString).Limit(pageSize).Offset(pageSize * page).Find(&payloads).Count(&count)

	return count, payloads
}
