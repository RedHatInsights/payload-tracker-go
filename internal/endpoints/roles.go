package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

// RolesArchiveLink returns a response for /roles/archiveLink
func RolesArchiveLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	statusCode, err := checkForRole(r, config.Get().StorageBrokerURLRole)
	if err != nil {
		writeResponse(w, statusCode, getErrorBody(fmt.Sprintf("%v", err), statusCode))
		return
	}

	allowed, _ := json.Marshal(
		structs.ArchiveLinkRole{
			Allowed: true,
		},
	)

	writeResponse(w, http.StatusOK, string(allowed))
}
