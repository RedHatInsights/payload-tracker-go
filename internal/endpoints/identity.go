package endpoints

import (
	"errors"
	"net/http"

	"github.com/redhatinsights/platform-go-middlewares/v2/identity"

	"github.com/redhatinsights/payload-tracker-go/internal/securitylog"
)

// associateIdentityType is the x-rh-identity type set by Turnpike for
// SAML-authenticated Red Hat associates.
const associateIdentityType = "Associate"

var (
	errMissingIdentity      = errors.New("Missing Identity Header")
	errInvalidIdentity      = errors.New("Invalid Identity Header")
	errNotAssociateIdentity = errors.New("This resource is only available to Red Hat associates")
)

// decodeAssociateIdentity decodes the x-rh-identity header and requires it to be
// an Associate identity as issued by Turnpike. It returns the decoded identity,
// or an HTTP status code and error describing why the request must be rejected.
//
// Payload Tracker is an internal tool: associates are allowed to read data for
// any org, so results are intentionally not scoped by the caller's org_id.
func decodeAssociateIdentity(r *http.Request, resourceType string) (identity.XRHID, int, error) {
	identityHeader := r.Header.Get("x-rh-identity")
	principal := securitylog.PrincipalFromRequest(r)

	if identityHeader == "" {
		securitylog.LogAuthFailure("missing identity header", resourceType, "", principal)
		return identity.XRHID{}, http.StatusUnauthorized, errMissingIdentity
	}

	id, err := identity.DecodeIdentity(identityHeader)
	if err != nil {
		securitylog.LogAuthFailure("invalid identity header", resourceType, "", principal)
		return identity.XRHID{}, http.StatusUnauthorized, errInvalidIdentity
	}

	if id.Identity.Type != associateIdentityType || id.Identity.Associate == nil {
		securitylog.LogAuthzFailure("identity type not allowed: "+id.Identity.Type, resourceType, "", principal)
		return identity.XRHID{}, http.StatusForbidden, errNotAssociateIdentity
	}

	return id, http.StatusOK, nil
}

// EnforceAssociateIdentity is a middleware that rejects requests that do not
// carry a valid Turnpike-issued Associate x-rh-identity header. On success the
// decoded identity is placed in the request context (see identity.GetIdentity).
func EnforceAssociateIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, statusCode, err := decodeAssociateIdentity(r, "api")
		if err != nil {
			writeResponse(w, statusCode, getErrorBody(err.Error(), statusCode))
			return
		}

		ctx := identity.WithIdentity(r.Context(), id)
		ctx = identity.WithRawIdentity(ctx, r.Header.Get("x-rh-identity"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
