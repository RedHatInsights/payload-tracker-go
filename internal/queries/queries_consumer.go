package queries

import (
	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusColumns = "payload_id, status_id, service_id, source_id, date, inventory_id, system_id, account, org_id"
	PayloadJoins  = "left join Payloads on Payloads.id = PayloadStatuses.payload_id"
)

var (
	services []models.Services
	statuses []models.Statuses
	sources  []models.Sources
)

func GetServiceByName(db *gorm.DB, service_id string) models.Services {
	var service models.Services
	db.Where("name = ?", service_id).First(&service)
	return service
}

func GetStatusByName(db *gorm.DB, status_id string) models.Statuses {
	var status models.Statuses
	db.Where("name = ?", status_id).First(&status)
	return status
}

func GetSourceByName(db *gorm.DB, source_id string) models.Sources {
	var source models.Sources
	db.Where("name = ?", source_id).First(&source)
	return source
}

func GetPayloadByRequestId(db *gorm.DB, request_id string) (result models.Payloads, err error) {
	var payload models.Payloads
	if results := db.Where("request_id = ?", request_id).First(&payload); results.Error != nil {
		return payload, results.Error
	}

	return payload, nil
}

func UpsertPayloadByRequestId(db *gorm.DB, request_id string, payload models.Payloads) (tx *gorm.DB, payloadId uint) {
	columnsToUpdate := []string{"request_id"}

	if payload.Account != "" {
		columnsToUpdate = append(columnsToUpdate, "account")
	}
	if payload.OrgId != "" {
		columnsToUpdate = append(columnsToUpdate, "org_id")
	}
	if payload.InventoryId != "" {
		columnsToUpdate = append(columnsToUpdate, "inventory_id")
	}
	if payload.SystemId != "" {
		columnsToUpdate = append(columnsToUpdate, "system_id")
	}

	onConflict := clause.OnConflict{
		Columns: []clause.Column{{Name: "request_id"}},
		DoUpdates: clause.AssignmentColumns(columnsToUpdate),
	}

	result := db.Model(&payload).Clauses(onConflict).Create(&payload)

	return result, payload.Id
}

func UpdatePayloadsTable(db *gorm.DB, updates models.Payloads, payloads models.Payloads) (tx *gorm.DB) {
	return db.Model(&payloads).Omit("request_id", "Id").Updates(updates)
}

func CreatePayloadTableEntry(db *gorm.DB, newPayload models.Payloads) (result *gorm.DB, payload models.Payloads) {
	results := db.Create(&newPayload)

	return results, newPayload
}

func CreateStatusTableEntry(db *gorm.DB, name string) (result *gorm.DB, status models.Statuses) {
	newStatus := models.Statuses{Name: name}
	results := db.Create(&newStatus)

	return results, newStatus
}

func CreateSourceTableEntry(db *gorm.DB, name string) (result *gorm.DB, source models.Sources) {
	newSource := models.Sources{Name: name}
	results := db.Create(&newSource)

	return results, newSource
}

func CreateServiceTableEntry(db *gorm.DB, name string) (result *gorm.DB, service models.Services) {
	newService := models.Services{Name: name}
	results := db.Create(&newService)

	return results, newService
}

func InsertPayloadStatus(db *gorm.DB, payloadStatus *models.PayloadStatuses) (tx *gorm.DB) {
	if (models.Sources{}) == payloadStatus.Source {
		return db.Omit("source_id").Create(&payloadStatus)
	}
	return db.Create(&payloadStatus)
}
