package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redhatinsights/payload-tracker-go/internal/queries"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	RetrieveStatuses = queries.RetrieveStatuses
)

func Statuses(w http.ResponseWriter, r *http.Request) {
	// init query with defaults and passed params
	start := time.Now()

	q, err := initQuery(r)

	if err != nil {
		writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%v", err), http.StatusBadRequest))
		return
	}

	if !stringInSlice(q.SortBy, validStatusesSortBy) {
		message := "sort_by must be one of " + strings.Join(validStatusesSortBy, ", ")
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}
	if !stringInSlice(q.SortDir, validSortDir) {
		message := "sort_dir must be one of " + strings.Join(validSortDir, ", ")
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}
	if !validTimestamps(q, true) {
		message := "invalid timestamp format provided"
		writeResponse(w, http.StatusBadRequest, getErrorBody(message, http.StatusBadRequest))
		return
	}
	count, payloads := RetrieveStatuses(getDb(), q)
	duration := time.Since(start).Seconds()

	statusesData := structs.StatusesData{count, duration, payloads}

	dataJson, err := json.Marshal(statusesData)
	if err != nil {
		l.Log.Error(err)
		writeResponse(w, http.StatusInternalServerError, getErrorBody("Internal Server Issue", http.StatusInternalServerError))
		return
	}

	writeResponse(w, http.StatusOK, string(dataJson))
}
