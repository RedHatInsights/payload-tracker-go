package db_methods

import (
	"fmt"
	// "encoding/json"
	"math"
	"strings"
	// "time"

	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	payloadsFields        = []string{"payloads.id", "payloads.request_id", "payloads.account", "payloads.request_id", "payloads.system_id"}
	payloadStatusesFields = []string{"payload_statuses.status_msg", "payload_statuses.date", "payload_statuses.created_at"}
	otherFields           = []string{"services.name as service", "sources.name as source", "statuses.name as status"}
)

func arrayMinMax(numArr []int64) (int64, int64) {
	min := int64(math.MaxInt64)
	max := int64(math.MinInt64)

	for _, v := range numArr {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

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

func RetrieveRequestIdPayloads(reqID string, sortBy string, sortDir string) []structs.SinglePayloadData {
	var payloads []structs.SinglePayloadData

	dbQuery := db.DB

	fields := strings.Join(payloadsFields, ",") + "," + strings.Join(payloadStatusesFields, ",") + "," + strings.Join(otherFields, ",")
	dbQuery = dbQuery.Table("payload_statuses").Select(fields).Joins("JOIN payloads on payload_statuses.payload_id = payloads.id")
	dbQuery = dbQuery.Joins("JOIN services on payload_statuses.service_id = services.id").Joins("JOIN sources on payload_statuses.source_id = sources.id").Joins("JOIN statuses on payload_statuses.status_id = statuses.id")

	orderString := fmt.Sprintf("%s %s", sortBy, sortDir)

	dbQuery.Where("payloads.request_id = ?", reqID).Order(orderString).Scan(&payloads)
	fmt.Println("Request_id", reqID, orderString)
	return payloads
}

func CalculateDurations(payloadData []structs.SinglePayloadData) map[string]string {
	//service:source

	mapTimeArray := make(map[string][]int64)
	mapTimeString := make(map[string]string)

	serviceSource := ""
	service := ""
	source := "undefined"

	for _, v := range payloadData {
		seconds := v.Date.Unix()

		service = v.Service
		if v.Source != "" {
			source = v.Source
		}

		serviceSource = fmt.Sprintf("%s:%s", service, source)

		if array, ok := mapTimeArray[serviceSource]; !ok {
			mapTimeArray[serviceSource] = []int64{seconds}
		} else {
			mapTimeArray[serviceSource] = append(array, seconds)
		}

		fmt.Println(mapTimeArray)
	}

	for key, timeArray := range mapTimeArray {
		min, max := arrayMinMax(timeArray)
		// duration := max.Sub(min)
		duration := max - min
		fmt.Println(key, duration)
	}

	return mapTimeString
}
