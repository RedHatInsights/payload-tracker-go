package db_methods

import (
	"fmt"
	"strings"
	"time"

	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	payloadsFields        = []string{"payloads.id", "payloads.request_id", "payloads.account", "payloads.request_id", "payloads.system_id"}
	payloadStatusesFields = []string{"payload_statuses.status_msg", "payload_statuses.date", "payload_statuses.created_at"}
	otherFields           = []string{"services.name as service", "sources.name as source", "statuses.name as status"}
)

func interpretDuration(duration int64) string {
	rem := duration

	h := duration / int64(time.Hour)
	rem = rem - h*int64(time.Hour)

	m := rem / int64(time.Minute)
	rem = rem - m*int64(time.Minute)

	s := float64(rem) / float64(time.Second)

	strTest := fmt.Sprintf("%02d:%02d:%09.6f", h, m, s)
	return strTest
}

func updateMinMax(unixTime int64, store [2]int64) [2]int64 {
	if unixTime < store[0] {
		store[0] = unixTime
	} else if unixTime > store[1] {
		store[1] = unixTime
	}
	return store
}

var RetrievePayloads = func(page int, pageSize int, apiQuery structs.Query) (int64, []models.Payloads) {
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

var RetrieveRequestIdPayloads = func(reqID string, sortBy string, sortDir string) []structs.SinglePayloadData {
	var payloads []structs.SinglePayloadData

	dbQuery := db.DB

	fields := strings.Join(payloadsFields, ",") + "," + strings.Join(payloadStatusesFields, ",") + "," + strings.Join(otherFields, ",")
	dbQuery = dbQuery.Table("payload_statuses").Select(fields).Joins("JOIN payloads on payload_statuses.payload_id = payloads.id")
	dbQuery = dbQuery.Joins("JOIN services on payload_statuses.service_id = services.id").Joins("JOIN sources on payload_statuses.source_id = sources.id").Joins("JOIN statuses on payload_statuses.status_id = statuses.id")

	orderString := fmt.Sprintf("%s %s", sortBy, sortDir)

	dbQuery.Where("payloads.request_id = ?", reqID).Order(orderString).Scan(&payloads)

	return payloads
}

func CalculateDurations(payloadData []structs.SinglePayloadData) map[string]string {
	//service:source

	mapTimeArray := make(map[string][2]int64)
	mapTimeString := make(map[string]string)

	for _, v := range payloadData {
		serviceSource := ""
		service := ""
		source := "undefined"

		nanoSeconds := v.Date.UnixNano()

		service = v.Service
		if v.Source != "" {
			source = v.Source
		}

		serviceSource = fmt.Sprintf("%s:%s", service, source)

		if array, ok := mapTimeArray[serviceSource]; !ok {
			mapTimeArray[serviceSource] = [2]int64{nanoSeconds, nanoSeconds}
		} else {
			mapTimeArray[serviceSource] = updateMinMax(nanoSeconds, array)
		}
	}

	for key, timeArray := range mapTimeArray {
		min, max := timeArray[0], timeArray[1]
		duration := max - min
		mapTimeString[key] = interpretDuration(duration)
	}

	return mapTimeString
}
