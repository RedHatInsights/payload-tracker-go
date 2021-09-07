package db_methods

import (
	"fmt"

	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

func RetrievePayloads(page int, pageSize int, apiQuery structs.Query) []models.Payloads {
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

	orderString := fmt.Sprintf("%s %s", apiQuery.SortBy, apiQuery.SortDir)
	dbQuery.Order(orderString).Limit(pageSize).Offset(pageSize * page).Find(&payloads)

	fmt.Println("inventory_id", apiQuery.InventoryID)

	return payloads
}
