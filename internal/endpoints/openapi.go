package endpoints

import (
	"net/http"

	"github.com/redhatinsights/payload-tracker-go/api"
)

// OpenAPISpec serves the embedded OpenAPI 3.x contract (api/api.spec.json) as-is.
// Deliberately registered without any identity/RBAC check so external tools and
// docs generators can fetch it anonymously.
func OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, http.StatusOK, string(api.Spec))
}
