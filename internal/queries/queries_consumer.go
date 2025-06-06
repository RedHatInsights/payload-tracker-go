package queries

import (
	"fmt"
	"time"

	expirable_lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redhatinsights/payload-tracker-go/internal/config"
	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusColumns = "payload_id, status_id, service_id, source_id, date, inventory_id, system_id, account, org_id"
	PayloadJoins  = "left join Payloads on Payloads.id = PayloadStatuses.payload_id"
)

type PayloadFieldsRepository interface {
	GetStatus(string) models.Statuses
	GetService(string) models.Services
	GetSource(string) models.Sources
}

type PayloadFieldsRepositoryFromDB struct {
	DB *gorm.DB
}

type PayloadFieldsRepositoryFromCache struct {
	PayloadFields PayloadFieldsRepository
	statusCache   *expirable_lru.LRU[string, models.Statuses]
	serviceCache  *expirable_lru.LRU[string, models.Services]
	sourceCache   *expirable_lru.LRU[string, models.Sources]
}

func (p *PayloadFieldsRepositoryFromDB) GetStatus(statusName string) models.Statuses {
	var status models.Statuses

	p.DB.Where("name = ?", statusName).First(&status)
	return status
}

func (p *PayloadFieldsRepositoryFromDB) GetService(serviceName string) models.Services {
	var service models.Services

	p.DB.Where("name = ?", serviceName).First(&service)
	return service
}

func (p *PayloadFieldsRepositoryFromDB) GetSource(sourceName string) models.Sources {
	var source models.Sources

	p.DB.Where("name = ?", sourceName).First(&source)
	return source
}

func (p *PayloadFieldsRepositoryFromCache) GetStatus(statusName string) models.Statuses {
	cached, ok := p.statusCache.Get(statusName)
	if ok {
		fmt.Printf("status cache hit: %+v\n", cached)
		return cached
	}

	dbEntry := p.PayloadFields.GetStatus(statusName)

	p.statusCache.Add(statusName, dbEntry)

	fmt.Printf("status cache miss: %+v\n", dbEntry)

	return dbEntry
}

func (p *PayloadFieldsRepositoryFromCache) GetService(serviceName string) models.Services {
	cached, ok := p.serviceCache.Get(serviceName)
	if ok {
		fmt.Printf("service cache hit: %+v\n", cached)
		return cached
	}

	dbEntry := p.PayloadFields.GetService(serviceName)

	p.serviceCache.Add(serviceName, dbEntry)

	fmt.Printf("service cache miss: %+v\n", dbEntry)

	return dbEntry
}

func (p *PayloadFieldsRepositoryFromCache) GetSource(sourceName string) models.Sources {
	cached, ok := p.sourceCache.Get(sourceName)
	if ok {
		fmt.Printf("source cache hit: %+v\n", cached)
		return cached
	}

	dbEntry := p.PayloadFields.GetSource(sourceName)

	p.sourceCache.Add(sourceName, dbEntry)

	fmt.Printf("source cache miss: %+v\n", dbEntry)

	return dbEntry
}

func NewPayloadFieldsRepository(db *gorm.DB, cfg *config.TrackerConfig) (payloadFieldsRepository PayloadFieldsRepository, err error) {
	payloadDB := &PayloadFieldsRepositoryFromDB{DB: db}

	switch cfg.ConsumerConfig.ConsumerPayloadFieldsRepoImpl {
	case "db":
		return payloadDB, nil
	case "db_with_cache":
		payloadFieldsRepository, err = newPayloadFieldsRepositoryFromCache(payloadDB)
	default:
		return nil, fmt.Errorf("unable to configure PayloadFieldRepository implementation")
	}

	return payloadFieldsRepository, err
}

func newPayloadFieldsRepositoryFromCache(payloadFieldsRepository PayloadFieldsRepository) (PayloadFieldsRepository, error) {
	statusCache := expirable_lru.NewLRU[string, models.Statuses](0, nil, 12*time.Hour)
	if statusCache == nil {
		return nil, fmt.Errorf("unable to create LRU cache for caching Status results")
	}

	serviceCache := expirable_lru.NewLRU[string, models.Services](0, nil, 12*time.Hour)
	if serviceCache == nil {
		return nil, fmt.Errorf("unable to create LRU cache for caching Service results")
	}

	sourceCache := expirable_lru.NewLRU[string, models.Sources](0, nil, 12*time.Hour)
	if sourceCache == nil {
		return nil, fmt.Errorf("unable to create LRU cache for caching Source results")
	}

	return &PayloadFieldsRepositoryFromCache{
		PayloadFields: payloadFieldsRepository,
		statusCache:   statusCache,
		serviceCache:  serviceCache,
		sourceCache:   sourceCache,
	}, nil
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
		Columns:   []clause.Column{{Name: "request_id"}},
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
