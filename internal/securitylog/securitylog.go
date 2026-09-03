package securitylog

import (
	"net/http"

	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/sirupsen/logrus"

	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
)

// Principal represents the identity making a request.
type Principal struct {
	OrgID  string `json:"org_id"`
	UserID string `json:"user_id"`
	Type   string `json:"type"`
}

// PrincipalFromRequest extracts the principal from the x-rh-identity header
// using the platform-go-middlewares identity package.
func PrincipalFromRequest(r *http.Request) Principal {
	header := r.Header.Get("x-rh-identity")

	id, err := identity.DecodeIdentity(header)
	if err != nil {
		return Principal{Type: "anonymous"}
	}

	p := Principal{
		OrgID: id.Identity.OrgID,
		Type:  id.Identity.Type,
	}

	switch {
	case id.Identity.User != nil && id.Identity.User.UserID != "":
		p.UserID = id.Identity.User.UserID
		if p.Type == "" {
			p.Type = "User"
		}
	case id.Identity.Associate != nil && id.Identity.Associate.Email != "":
		p.UserID = id.Identity.Associate.Email
		if p.Type == "" {
			p.Type = "Associate"
		}
	case id.Identity.ServiceAccount != nil && id.Identity.ServiceAccount.ClientId != "":
		p.UserID = id.Identity.ServiceAccount.ClientId
		if p.Type == "" {
			p.Type = "ServiceAccount"
		}
	default:
		if p.Type == "" {
			p.Type = "unknown"
		}
	}

	if p.OrgID == "" {
		p.OrgID = id.Identity.AccountNumber
	}

	return p
}

// fields returns the common structured fields for all security events.
func fields(action, resourceType, resourceID, outcome string, principal Principal) logrus.Fields {
	return logrus.Fields{
		"security_event": true,
		"action":         action,
		"resource_type":  resourceType,
		"resource_id":    resourceID,
		"outcome":        outcome,
		"org_id":         principal.OrgID,
		"user_id":        principal.UserID,
		"principal_type": principal.Type,
	}
}

// Log emits a security event at Info level.
func Log(action, resourceType, resourceID, outcome string, principal Principal) {
	l.Log.WithFields(fields(action, resourceType, resourceID, outcome, principal)).
		Info("security event")
}

// LogAuthFailure emits an authentication failure security event at Warn level.
func LogAuthFailure(reason, resourceType, resourceID string, principal Principal) {
	l.Log.WithFields(fields("AUTH_FAILURE", resourceType, resourceID, "failure", principal)).
		WithField("reason", reason).
		Warn("security event: authentication failure")
}

// LogAuthzFailure emits an authorization failure security event at Warn level.
func LogAuthzFailure(reason, resourceType, resourceID string, principal Principal) {
	l.Log.WithFields(fields("AUTHZ_FAILURE", resourceType, resourceID, "failure", principal)).
		WithField("reason", reason).
		Warn("security event: authorization failure")
}

// LogLifecycle emits a lifecycle security event (STARTUP/SHUTDOWN) at Info level.
func LogLifecycle(action, outcome, component string) {
	l.Log.WithFields(logrus.Fields{
		"security_event": true,
		"action":         action,
		"resource_type":  "service",
		"resource_id":    component,
		"outcome":        outcome,
		"org_id":         "",
		"user_id":        "",
		"principal_type": "system",
	}).Info("security event: lifecycle")
}
