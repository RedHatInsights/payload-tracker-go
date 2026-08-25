package endpoints

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	validSortBy         = []string{"created_at", "account", "org_id", "system_id", "inventory_id", "service", "source", "status_msg", "date", "request_id", "status"}
	validAllSortBy      = []string{"account", "org_id", "inventory_id", "system_id", "created_at"}
	validIDSortBy       = []string{"service", "source", "status_msg", "date", "created_at"}
	validStatusesSortBy = []string{"service", "source", "request_id", "status", "status_msg", "date", "created_at"}
	validSortDir        = []string{"asc", "desc"}
)

// initQuery intializes the query with default values
func initQuery(r *http.Request) (structs.Query, error) {

	q := structs.Query{
		Page:         0,
		PageSize:     10,
		SortBy:       "date",
		SortDir:      "desc",
		InventoryID:  r.URL.Query().Get("inventory_id"),
		SystemID:     r.URL.Query().Get("system_id"),
		CreatedAtLT:  r.URL.Query().Get("created_at_lt"),
		CreatedAtGT:  r.URL.Query().Get("created_at_gt"),
		CreatedAtLTE: r.URL.Query().Get("created_at_lte"),
		CreatedAtGTE: r.URL.Query().Get("created_at_gte"),
		Account:      r.URL.Query().Get("account"),
		OrgID:        r.URL.Query().Get("org_id"),

		Service:   r.URL.Query().Get("service"),
		Source:    r.URL.Query().Get("source"),
		Status:    r.URL.Query().Get("status"),
		StatusMsg: r.URL.Query().Get("status_msg"),
		DateLT:    r.URL.Query().Get("date_lt"),
		DateLTE:   r.URL.Query().Get("date_lte"),
		DateGT:    r.URL.Query().Get("date_gt"),
		DateGTE:   r.URL.Query().Get("date_gte"),
	}

	var err error

	if r.URL.Query().Get("sort_by") != "" || stringInSlice(r.URL.Query().Get("sort_by"), validSortBy) {
		q.SortBy = r.URL.Query().Get("sort_by")
	}

	if r.URL.Query().Get("sort_dir") != "" || stringInSlice(r.URL.Query().Get("sort_dir"), validSortDir) {
		q.SortDir = r.URL.Query().Get("sort_dir")
	}

	if r.URL.Query().Get("page") != "" {
		q.Page, err = strconv.Atoi(r.URL.Query().Get("page"))
	}

	if r.URL.Query().Get("page_size") != "" {
		q.PageSize, err = strconv.Atoi(r.URL.Query().Get("page_size"))
	}

	return q, err
}

func getDb() *gorm.DB {
	return db.DB
}

func getErrorBody(message string, status int) string {
	errBody := structs.ErrorResponse{
		Title:   http.StatusText(status),
		Message: message,
		Status:  status,
	}

	errBodyJson, _ := json.Marshal(errBody)
	return string(errBodyJson)
}

// hashIdentity returns a non-reversible short reference for an identity header
// so denied/audit log lines can be correlated without leaking associate PII.
func hashIdentity(identityHeader string) string {
	if identityHeader == "" {
		return "none"
	}
	sum := sha256.Sum256([]byte(identityHeader))
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(sum[:])[:16])
}

// Check for value in a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Check timestamp format
func validTimestamps(q structs.Query, all bool) bool {
	timestampQueries := []string{q.CreatedAtLT, q.CreatedAtGT, q.CreatedAtLTE, q.CreatedAtGTE, q.DateLT, q.DateGT, q.DateLTE, q.DateGTE}

	if !all {
		timestampQueries = []string{q.CreatedAtLT, q.CreatedAtGT, q.CreatedAtLTE, q.CreatedAtGTE}
	}

	for _, ts := range timestampQueries {
		if ts != "" {
			_, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return false
			}
		}
	}
	return true
}

// Check for a specified role in the user's identity header, returns (200, nil) if the role is found
func checkForRole(r *http.Request, role string) (int, error) {
	identityHeader := r.Header.Get("x-rh-identity")
	if identityHeader == "" {
		return http.StatusUnauthorized, errors.New("Missing Identity Header")
	}

	type IdentityHeader struct {
		Identity struct {
			Associate struct {
				Roles []string `json:"Role"`
			} `json:"associate"`
		} `json:"identity"`
	}

	var identityHeaderData IdentityHeader
	// base64 decode the header
	decoded, err := base64.StdEncoding.DecodeString(identityHeader)
	if err != nil {
		l.Log.Error("Error decoding identity header", "error", err)
		return http.StatusUnauthorized, err
	}

	err = json.Unmarshal(decoded, &identityHeaderData)
	if err != nil {
		l.Log.Error("Error unmarshalling identity header", "error", err)
		return http.StatusUnauthorized, err

	}

	if !stringInSlice(role, identityHeaderData.Identity.Associate.Roles) {
		l.Log.WithFields(logrus.Fields{
			"role":          role,
			"identity_hash": hashIdentity(identityHeader),
		}).Infof("Unable to find required role")
		return http.StatusForbidden, errors.New("You do not have the required permissions to access this resource")
	}

	return http.StatusOK, nil
}

// Write HTTP Response
func writeResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(message))
}

// Send a request for an ArchiveLink to storage-broker
func RequestArchiveLink(baseUrl string, timeout int) func(ctx context.Context, reqID string) (*structs.PayloadArchiveLink, error) {

	return func(ctx context.Context, reqID string) (*structs.PayloadArchiveLink, error) {
		client := http.Client{
			Timeout: time.Duration(timeout) * time.Millisecond,
		}

		response, err := client.Get(baseUrl + "?request_id=" + reqID)
		if err != nil {
			return nil, err
		}

		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		var archiveLink structs.PayloadArchiveLink
		err = json.Unmarshal(body, &archiveLink)
		if err != nil {
			return nil, err
		}

		return &archiveLink, nil
	}
}

func isValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	reqId := chi.URLParam(r, "id")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("You got an archive for " + reqId))
	return
}
