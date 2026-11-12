package endpoints

import (
	"errors"
	"net/http"

	"github.com/redhatinsights/platform-go-middlewares/v2/identity"

	"github.com/redhatinsights/payload-tracker-go/internal/securitylog"
)

// Identity types issued by Turnpike: "Associate" for SAML-authenticated Red Hat
// associates and "X509" for mTLS client-certificate authentication.
const (
	associateIdentityType = "Associate"
	x509IdentityType      = "X509"
)

var (
	errMissingIdentity      = errors.New("Missing Identity Header")
	errInvalidIdentity      = errors.New("Invalid Identity Header")
	errNotTurnpikeIdentity  = errors.New("This resource is only available through Turnpike")
	errNotAssociateIdentity = errors.New("This resource is only available to Red Hat associates")
)

// decodeIdentity decodes the x-rh-identity header. It returns the decoded
// identity, or an HTTP status code and error describing why the request must be
// rejected.
func decodeIdentity(r *http.Request, resourceType string) (identity.XRHID, int, error) {
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

	return id, http.StatusOK, nil
}

func isAssociateIdentity(id identity.XRHID) bool {
	return id.Identity.Type == associateIdentityType && id.Identity.Associate != nil
}

func isX509Identity(id identity.XRHID) bool {
	return id.Identity.Type == x509IdentityType && id.Identity.X509 != nil && id.Identity.X509.SubjectDN != ""
}

// decodeTurnpikeIdentity requires the x-rh-identity header to be an Associate
// or X509 identity, the two types Turnpike issues for the payload-tracker route.
//
// Payload Tracker is an internal tool: these callers are allowed to read data
// for any org, so results are intentionally not scoped by org_id.
func decodeTurnpikeIdentity(r *http.Request, resourceType string) (identity.XRHID, int, error) {
	id, statusCode, err := decodeIdentity(r, resourceType)
	if err != nil {
		return id, statusCode, err
	}

	if !isAssociateIdentity(id) && !isX509Identity(id) {
		securitylog.LogAuthzFailure("identity type not allowed: "+id.Identity.Type, resourceType, "", securitylog.PrincipalFromRequest(r))
		return identity.XRHID{}, http.StatusForbidden, errNotTurnpikeIdentity
	}

	return id, http.StatusOK, nil
}

// decodeAssociateIdentity requires the x-rh-identity header to be an Associate
// identity. Used where LDAP roles (identity.associate.Role) are checked.
func decodeAssociateIdentity(r *http.Request, resourceType string) (identity.XRHID, int, error) {
	id, statusCode, err := decodeIdentity(r, resourceType)
	if err != nil {
		return id, statusCode, err
	}

	if !isAssociateIdentity(id) {
		securitylog.LogAuthzFailure("identity type not allowed: "+id.Identity.Type, resourceType, "", securitylog.PrincipalFromRequest(r))
		return identity.XRHID{}, http.StatusForbidden, errNotAssociateIdentity
	}

	return id, http.StatusOK, nil
}

// EnforceValidIdentity is a middleware that requires a well-formed
// x-rh-identity header of any type that passes the platform base policy
// (identity.DecodeAndCheckIdentity: an org_id is required unless the type is
// Associate or X509). It is used on /payloads/{request_id}, which the ingress
// service's customer-facing /track endpoint calls with the caller's forwarded
// identity; ingress performs the org_id authorization. On success the decoded
// identity is placed in the request context.
func EnforceValidIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identityHeader := r.Header.Get("x-rh-identity")
		principal := securitylog.PrincipalFromRequest(r)

		if identityHeader == "" {
			securitylog.LogAuthFailure("missing identity header", "payload", "", principal)
			writeResponse(w, http.StatusUnauthorized, getErrorBody(errMissingIdentity.Error(), http.StatusUnauthorized))
			return
		}

		id, err := identity.DecodeAndCheckIdentity(identityHeader)
		if err != nil {
			securitylog.LogAuthFailure("invalid identity header: "+err.Error(), "payload", "", principal)
			writeResponse(w, http.StatusUnauthorized, getErrorBody(errInvalidIdentity.Error(), http.StatusUnauthorized))
			return
		}

		ctx := identity.WithIdentity(r.Context(), id)
		ctx = identity.WithRawIdentity(ctx, identityHeader)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnforceTurnpikeIdentity is a middleware that rejects requests that do not
// carry a valid Turnpike-issued (Associate or X509) x-rh-identity header. On
// success the decoded identity is placed in the request context (see
// identity.GetIdentity).
func EnforceTurnpikeIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, statusCode, err := decodeTurnpikeIdentity(r, "api")
		if err != nil {
			writeResponse(w, statusCode, getErrorBody(err.Error(), statusCode))
			return
		}

		ctx := identity.WithIdentity(r.Context(), id)
		ctx = identity.WithRawIdentity(ctx, r.Header.Get("x-rh-identity"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
