package db_methods

import (
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
)

func RetrievePayloads(page int, pageSize int) []models.Payloads {
	var payloads []models.Payloads

	db.DB.Limit(pageSize).Offset(pageSize * page).Find(&payloads)

	return payloads
}
