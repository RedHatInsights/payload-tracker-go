package securitylog

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/redhatinsights/payload-tracker-go/internal/logging"
)

// captureOutput sets up logging to capture output and returns a function to
// retrieve the captured JSON fields.
func captureOutput(t *testing.T) (*bytes.Buffer, func() map[string]interface{}) {
	t.Helper()

	buf := &bytes.Buffer{}
	logging.Log = &logrus.Logger{
		Out:          buf,
		Level:        logrus.DebugLevel,
		Formatter:    &logrus.JSONFormatter{},
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: false,
	}

	return buf, func() map[string]interface{} {
		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse log output: %v\nraw: %s", err, buf.String())
		}
		return result
	}
}

func makeIdentityHeader(orgID, userID, identityType string) string {
	id := map[string]interface{}{
		"identity": map[string]interface{}{
			"org_id": orgID,
			"type":   identityType,
			"user": map[string]interface{}{
				"user_id": userID,
			},
		},
	}
	b, _ := json.Marshal(id)
	return base64.StdEncoding.EncodeToString(b)
}

func makeAssociateIdentityHeader(orgID, email string) string {
	id := map[string]interface{}{
		"identity": map[string]interface{}{
			"org_id": orgID,
			"type":   "Associate",
			"associate": map[string]interface{}{
				"email": email,
			},
		},
	}
	b, _ := json.Marshal(id)
	return base64.StdEncoding.EncodeToString(b)
}

func makeServiceAccountHeader(orgID, clientID string) string {
	id := map[string]interface{}{
		"identity": map[string]interface{}{
			"org_id": orgID,
			"type":   "ServiceAccount",
			"service_account": map[string]interface{}{
				"client_id": clientID,
			},
		},
	}
	b, _ := json.Marshal(id)
	return base64.StdEncoding.EncodeToString(b)
}

func TestPrincipalFromRequest_User(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("x-rh-identity", makeIdentityHeader("org123", "user456", "User"))

	p := PrincipalFromRequest(req)

	if p.OrgID != "org123" {
		t.Errorf("expected org_id=org123, got %s", p.OrgID)
	}
	if p.UserID != "user456" {
		t.Errorf("expected user_id=user456, got %s", p.UserID)
	}
	if p.Type != "User" {
		t.Errorf("expected type=User, got %s", p.Type)
	}
}

func TestPrincipalFromRequest_Associate(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("x-rh-identity", makeAssociateIdentityHeader("org789", "user@redhat.com"))

	p := PrincipalFromRequest(req)

	if p.OrgID != "org789" {
		t.Errorf("expected org_id=org789, got %s", p.OrgID)
	}
	if p.UserID != "user@redhat.com" {
		t.Errorf("expected user_id=user@redhat.com, got %s", p.UserID)
	}
	if p.Type != "Associate" {
		t.Errorf("expected type=Associate, got %s", p.Type)
	}
}

func TestPrincipalFromRequest_ServiceAccount(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("x-rh-identity", makeServiceAccountHeader("org111", "client-abc"))

	p := PrincipalFromRequest(req)

	if p.OrgID != "org111" {
		t.Errorf("expected org_id=org111, got %s", p.OrgID)
	}
	if p.UserID != "client-abc" {
		t.Errorf("expected user_id=client-abc, got %s", p.UserID)
	}
	if p.Type != "ServiceAccount" {
		t.Errorf("expected type=ServiceAccount, got %s", p.Type)
	}
}

func TestPrincipalFromRequest_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	p := PrincipalFromRequest(req)

	if p.Type != "anonymous" {
		t.Errorf("expected type=anonymous, got %s", p.Type)
	}
}

func TestPrincipalFromRequest_InvalidBase64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("x-rh-identity", "not-valid-base64!!!")

	p := PrincipalFromRequest(req)

	if p.Type != "anonymous" {
		t.Errorf("expected type=anonymous, got %s", p.Type)
	}
}

func TestPrincipalFromRequest_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("x-rh-identity", base64.StdEncoding.EncodeToString([]byte("{invalid")))

	p := PrincipalFromRequest(req)

	if p.Type != "anonymous" {
		t.Errorf("expected type=anonymous, got %s", p.Type)
	}
}

func TestLog(t *testing.T) {
	_, getFields := captureOutput(t)

	principal := Principal{OrgID: "org1", UserID: "user1", Type: "User"}
	Log("READ", "payload", "req-123", "success", principal)

	f := getFields()
	if f["security_event"] != true {
		t.Error("expected security_event=true")
	}
	if f["action"] != "READ" {
		t.Errorf("expected action=READ, got %v", f["action"])
	}
	if f["resource_type"] != "payload" {
		t.Errorf("expected resource_type=payload, got %v", f["resource_type"])
	}
	if f["resource_id"] != "req-123" {
		t.Errorf("expected resource_id=req-123, got %v", f["resource_id"])
	}
	if f["outcome"] != "success" {
		t.Errorf("expected outcome=success, got %v", f["outcome"])
	}
	if f["org_id"] != "org1" {
		t.Errorf("expected org_id=org1, got %v", f["org_id"])
	}
	if f["user_id"] != "user1" {
		t.Errorf("expected user_id=user1, got %v", f["user_id"])
	}
}

func TestLogAuthFailure(t *testing.T) {
	_, getFields := captureOutput(t)

	principal := Principal{Type: "anonymous"}
	LogAuthFailure("missing identity header", "archive_link", "", principal)

	f := getFields()
	if f["security_event"] != true {
		t.Error("expected security_event=true")
	}
	if f["action"] != "AUTH_FAILURE" {
		t.Errorf("expected action=AUTH_FAILURE, got %v", f["action"])
	}
	if f["reason"] != "missing identity header" {
		t.Errorf("expected reason='missing identity header', got %v", f["reason"])
	}
	if f["level"] != "warning" {
		t.Errorf("expected level=warning, got %v", f["level"])
	}
}

func TestLogAuthzFailure(t *testing.T) {
	_, getFields := captureOutput(t)

	principal := Principal{OrgID: "org1", UserID: "user1", Type: "Associate"}
	LogAuthzFailure("role not found", "archive_link", "req-456", principal)

	f := getFields()
	if f["security_event"] != true {
		t.Error("expected security_event=true")
	}
	if f["action"] != "AUTHZ_FAILURE" {
		t.Errorf("expected action=AUTHZ_FAILURE, got %v", f["action"])
	}
	if f["reason"] != "role not found" {
		t.Errorf("expected reason='role not found', got %v", f["reason"])
	}
}

func TestLogLifecycle(t *testing.T) {
	_, getFields := captureOutput(t)

	LogLifecycle("STARTUP", "success", "pt-api")

	f := getFields()
	if f["security_event"] != true {
		t.Error("expected security_event=true")
	}
	if f["action"] != "STARTUP" {
		t.Errorf("expected action=STARTUP, got %v", f["action"])
	}
	if f["resource_type"] != "service" {
		t.Errorf("expected resource_type=service, got %v", f["resource_type"])
	}
	if f["resource_id"] != "pt-api" {
		t.Errorf("expected resource_id=pt-api, got %v", f["resource_id"])
	}
	if f["outcome"] != "success" {
		t.Errorf("expected outcome=success, got %v", f["outcome"])
	}
	if f["principal_type"] != "system" {
		t.Errorf("expected principal_type=system, got %v", f["principal_type"])
	}
}
