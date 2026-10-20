package queries

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	payloadFields         = []string{"payloads.id", "payloads.request_id"}
	extraPayloadFields    = []string{"payloads.account", "payloads.org_id", "payloads.system_id", "payloads.inventory_id"}
	payloadStatusesFields = []string{"payload_statuses.status_msg", "payload_statuses.date", "payload_statuses.created_at"}
	otherFields           = []string{"services.name as service", "sources.name as source", "statuses.name as status"}
)

func defineVerbosity(verbosity string) string {
	switch verbosity {
	case "1":
		queryFields := []string{otherFields[0], otherFields[2], extraPayloadFields[3], payloadStatusesFields[1], payloadStatusesFields[0]}
		return strings.Join(queryFields, ",")
	case "2":
		queryFields := []string{otherFields[0], otherFields[2], payloadStatusesFields[1]}
		return strings.Join(queryFields, ",")
	default:
		queryFields := fmt.Sprintf("%s,%s,%s,%s", strings.Join(payloadFields, ","), strings.Join(extraPayloadFields, ","), strings.Join(payloadStatusesFields, ","), strings.Join(otherFields, ","))
		return queryFields
	}
}

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

func chainTimeConditions(dbColumn string, apiQuery structs.Query, dbQuery *gorm.DB) *gorm.DB {
	timeFields := map[string]string{
		"lt":  apiQuery.CreatedAtLT,
		"lte": apiQuery.CreatedAtLTE,
		"gt":  apiQuery.CreatedAtGT,
		"gte": apiQuery.CreatedAtGTE,
	}
	if dbColumn == "date" {
		timeFields = map[string]string{
			"lt":  apiQuery.DateLT,
			"lte": apiQuery.DateLTE,
			"gt":  apiQuery.DateGT,
			"gte": apiQuery.DateGTE,
		}
	}

	if timeFields["lt"] != "" {
		dbQuery = dbQuery.Where(fmt.Sprintf("%s < ?", dbColumn), timeFields["lt"])
	}
	if timeFields["lte"] != "" {
		dbQuery = dbQuery.Where(fmt.Sprintf("%s <= ?", dbColumn), timeFields["lte"])
	}
	if timeFields["gt"] != "" {
		dbQuery = dbQuery.Where(fmt.Sprintf("%s > ?", dbColumn), timeFields["gt"])
	}
	if timeFields["gte"] != "" {
		dbQuery = dbQuery.Where(fmt.Sprintf("%s >= ?", dbColumn), timeFields["gte"])
	}
	return dbQuery
}

var RetrievePayloads = func(dbQuery *gorm.DB, page int, pageSize int, apiQuery structs.Query) (int64, []models.Payloads) {
	var count int64
	var payloads []models.Payloads

	// query chaining
	if apiQuery.Account != "" {
		dbQuery = dbQuery.Where("account = ?", apiQuery.Account)
	}
	if apiQuery.OrgID != "" {
		dbQuery = dbQuery.Where("org_id = ?", apiQuery.OrgID)
	}
	if apiQuery.InventoryID != "" {
		dbQuery = dbQuery.Where("inventory_id = ?", apiQuery.InventoryID)
	}
	if apiQuery.SystemID != "" {
		dbQuery = dbQuery.Where("system_id = ?", apiQuery.SystemID)
	}

	dbQuery = chainTimeConditions("created_at", apiQuery, dbQuery)

	orderString := fmt.Sprintf("%s %s", apiQuery.SortBy, apiQuery.SortDir)

	//dbQuery.Model(&payloads).Count(&count)
	dbQuery.Order(orderString).Limit(pageSize).Offset(pageSize * page).Find(&payloads)

	return count, payloads
}

var RetrieveRequestIdPayloads = func(dbQuery *gorm.DB, reqID string, sortBy string, sortDir string, verbosity string) []structs.SinglePayloadData {
	var payloads []structs.SinglePayloadData

	fields := defineVerbosity(verbosity)

	dbQuery = dbQuery.Table("payload_statuses").Select(fields).Joins("JOIN payloads on payload_statuses.payload_id = payloads.id")
	dbQuery = dbQuery.Joins("JOIN services on payload_statuses.service_id = services.id").Joins("FULL OUTER JOIN sources on payload_statuses.source_id = sources.id").Joins("JOIN statuses on payload_statuses.status_id = statuses.id")

	orderString := fmt.Sprintf("%s %s", sortBy, sortDir)

	dbQuery.Where("payloads.request_id = ?", reqID).Order(orderString).Scan(&payloads)

	return payloads
}

var RetrieveStatuses = func(dbQuery *gorm.DB, apiQuery structs.Query) (int64, []structs.StatusRetrieve) {
	var count int64
	var payloads []structs.StatusRetrieve

	page := apiQuery.Page
	pageSize := apiQuery.PageSize

	fields := fmt.Sprintf("%s,%s,%s", strings.Join(payloadFields, ","), strings.Join(payloadStatusesFields, ","), strings.Join(otherFields, ","))
	dbQuery = dbQuery.Table("payload_statuses").Select(fields).Joins("JOIN payloads on payload_statuses.payload_id = payloads.id")
	dbQuery = dbQuery.Joins("JOIN services on payload_statuses.service_id = services.id").Joins("JOIN sources on payload_statuses.source_id = sources.id").Joins("JOIN statuses on payload_statuses.status_id = statuses.id")

	// query chaining
	if apiQuery.Service != "" {
		dbQuery = dbQuery.Where("services.name = ?", apiQuery.Service)
	}
	if apiQuery.Source != "" {
		dbQuery = dbQuery.Where("sources.name = ?", apiQuery.Source)
	}
	if apiQuery.Status != "" {
		dbQuery = dbQuery.Where("statuses.name = ?", apiQuery.Status)
	}
	if apiQuery.StatusMsg != "" {
		dbQuery = dbQuery.Where("payload_statuses.status_msg = ?", apiQuery.StatusMsg)
	}
	dbQuery = chainTimeConditions("date", apiQuery, dbQuery)
	dbQuery = chainTimeConditions("payload_statuses.created_at", apiQuery, dbQuery)

	orderString := fmt.Sprintf("%s %s", apiQuery.SortBy, apiQuery.SortDir)
	//dbQuery.Model(&payloads).Count(&count)
	dbQuery.Order(orderString).Limit(pageSize).Offset(pageSize * page).Scan(&payloads)

	return count, payloads
}

func CalculateDurations(payloadData []structs.SinglePayloadData) map[string]string {
	//service:source

	mapTimeArray := make(map[string][2]int64)
	mapTimeString := make(map[string]string)
	mapDurations := make(map[string]int64)
	mapDurations["total_time_in_services"] = 0

	dateMinMaxArray := [2]int64{payloadData[0].Date.UnixNano(), payloadData[0].Date.UnixNano()}

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

		dateMinMaxArray = updateMinMax(nanoSeconds, dateMinMaxArray)
	}

	for key, timeArray := range mapTimeArray {
		min, max := timeArray[0], timeArray[1]
		duration := max - min
		mapDurations["total_time_in_services"] += duration
		mapTimeString[key] = interpretDuration(duration)
	}

	mapTimeString["total_time_in_services"] = interpretDuration(mapDurations["total_time_in_services"])
	mapTimeString["total_time"] = interpretDuration(dateMinMaxArray[1] - dateMinMaxArray[0])

	return mapTimeString
}
