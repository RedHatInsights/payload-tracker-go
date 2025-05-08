package kafka

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models/message"
	"github.com/redhatinsights/payload-tracker-go/internal/queries"
)

type DBInterface interface {
	GetStatusByName(db *gorm.DB, statusName string) models.Statuses
	GetServiceByName(db *gorm.DB, name string) models.Services
	GetSourceByName(db *gorm.DB, name string) models.Sources
}

type DBImpl struct {
	statusCache  map[string]models.Statuses
	serviceCache map[string]models.Services
	sourceCache  map[string]models.Sources
}

type handler struct {
	db *gorm.DB
}

func (h *handler) GetStatusByName(db *gorm.DB, statusName string) models.Statuses {
	cache := make(map[string]models.Statuses)

	cached, ok := cache[statusName]
	if ok {
		return cached
	}

	dataToCache := queries.GetDBStatusByName(db, statusName)

	cache[statusName] = dataToCache

	return dataToCache
}

func (h *handler) GetServiceByName(db *gorm.DB, serviceName string) models.Services {
	cache := make(map[string]models.Services)

	cached, ok := cache[serviceName]
	if ok {
		return cached
	}

	dataToCache := queries.GetDBServiceByName(db, serviceName)

	cache[serviceName] = dataToCache

	return dataToCache
}

func (h *handler) GetSourceByName(db *gorm.DB, sourceName string) models.Sources {
	cache := make(map[string]models.Sources)

	cached, ok := cache[sourceName]
	if ok {
		return cached
	}

	dataToCache := queries.GetDBSourceByName(db, sourceName)

	cache[sourceName] = dataToCache

	return dataToCache
}

// OnMessage takes in each payload status message and processes it
func (h *handler) onMessage(ctx context.Context, msg *kafka.Message, cfg *config.TrackerConfig) {
	// Track the time from beginning of handling the message to the insert
	start := time.Now()
	l.Log.Debug("Processing Payload Message ", string(msg.Value))

	payloadStatus := &message.PayloadStatusMessage{}
	sanitizedPayloadStatus := &models.PayloadStatuses{}

	if err := json.Unmarshal(msg.Value, payloadStatus); err != nil {
		// PROBE: Add probe here for error unmarshaling JSON
		if cfg.DebugConfig.LogStatusJson {
			l.Log.Error("ERROR: Unmarshaling Payload Status Event: ", err, " Raw Message: ", string(msg.Value))
		} else {
			l.Log.Error("ERROR: Unmarshaling Payload Status Event: ", err)
		}
		return
	}

	if !validateRequestID(cfg.RequestConfig.ValidateRequestIDLength, payloadStatus.RequestID) {
		return
	}

	// Sanitize the payload
	sanitizePayload(payloadStatus)

	// Upsert into Payloads Table
	payload := createPayload(payloadStatus)

	upsertResult, payloadId := queries.UpsertPayloadByRequestId(h.db, payloadStatus.RequestID, payload)
	if upsertResult.Error != nil {
		l.Log.Error("ERROR Payload table upsert failed: ", upsertResult.Error)
		return
	}
	sanitizedPayloadStatus.PayloadId = payloadId

	// Check if service/source/status are in table
	// this section checks the subsiquent DB tables to see if the service_id, source_id, and status_id exist for the given message
	l.Log.Debug("Adding Status, Sources, and Services to sanitizedPayload")

	// Status & Service: Always defined in the message
	var existingStatus models.Statuses
	existingStatus = h.GetStatusByName(h.db, payloadStatus.Status)

	var existingService models.Services
	existingService = h.GetServiceByName(h.db, payloadStatus.Service)

	if (models.Statuses{}) == existingStatus {
		statusResult, newStatus := queries.CreateStatusTableEntry(h.db, payloadStatus.Status)
		if statusResult.Error != nil {
			l.Log.Error("Error Creating Statuses Table Entry ERROR: ", statusResult.Error)
			return
		}

		sanitizedPayloadStatus.Status = newStatus
	} else {
		sanitizedPayloadStatus.Status = existingStatus
	}

	if (models.Services{}) == existingService {
		serviceResult, newService := queries.CreateServiceTableEntry(h.db, payloadStatus.Service)
		if serviceResult.Error != nil {
			l.Log.Error("Error Creating Service Table Entry ERROR: ", serviceResult.Error)
			return
		}

		sanitizedPayloadStatus.Service = newService
	} else {
		sanitizedPayloadStatus.Service = existingService
	}

	// Sources
	if payloadStatus.Source != "" {
		var existingSource models.Sources
		existingSource = h.GetSourceByName(h.db, payloadStatus.Source)

		if (models.Sources{}) == existingSource {
			result, newSource := queries.CreateSourceTableEntry(h.db, payloadStatus.Source)
			if result.Error != nil {
				l.Log.Error("Error Creating Sources Table Entry ERROR: ", result.Error)
				return
			}

			sanitizedPayloadStatus.Source = newSource
		} else {
			sanitizedPayloadStatus.Source = existingSource
		}
	}

	if payloadStatus.StatusMSG != "" {
		sanitizedPayloadStatus.StatusMsg = payloadStatus.StatusMSG
	}

	// Insert Date
	sanitizedPayloadStatus.Date = payloadStatus.Date.Time

	// Insert payload into DB
	endpoints.ObserveMessageProcessTime(time.Since(start))
	endpoints.IncMessagesProcessed()

	retries, attempts := cfg.DatabaseConfig.DBRetries, 0
	for retries > attempts {
		err := queries.InsertPayloadStatus(h.db, sanitizedPayloadStatus).Error

		if err == nil {
			break
		}

		l.Log.WithFields(logrus.Fields{"attempts": attempts}).Debug("Failed to insert sanitized PayloadStatus with ERROR: ", err)
		attempts += 1
	}
}

func validateRequestID(requestIDLength int, requestID string) bool {
	if requestIDLength != 0 {
		if len(requestID) != requestIDLength {
			endpoints.IncInvalidConsumerRequestIDs()
			return false
		}
	}

	return true
}

func sanitizePayload(msg *message.PayloadStatusMessage) {
	// Set default fields to lowercase
	msg.Service = strings.ToLower(msg.Service)
	msg.Status = strings.ToLower(msg.Status)
	if msg.Source != "" {
		msg.Source = strings.ToLower((msg.Source))
	}
}

func createPayload(msg *message.PayloadStatusMessage) (table models.Payloads) {
	payloadTable := models.Payloads{
		Id:          msg.PayloadID,
		RequestId:   msg.RequestID,
		Account:     msg.Account,
		OrgId:       msg.OrgID,
		SystemId:    msg.SystemID,
		CreatedAt:   msg.Date.Time,
		InventoryId: msg.InventoryID,
	}

	return payloadTable
}
