package endpoints_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

func encodeIdentity(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

var _ = Describe("EnforceTurnpikeIdentity", func() {
	var (
		handler     http.Handler
		rr          *httptest.ResponseRecorder
		nextCalled  bool
		ctxIdentity identity.XRHID
	)

	BeforeEach(func() {
		nextCalled = false
		ctxIdentity = identity.XRHID{}
		rr = httptest.NewRecorder()
		handler = endpoints.EnforceTurnpikeIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			ctxIdentity = identity.GetIdentity(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
	})

	serve := func(header string) {
		req, err := test.MakeTestRequest("/api/v1/payloads", map[string]interface{}{"page": 0})
		Expect(err).To(BeNil())
		if header != "" {
			req.Header.Set("x-rh-identity", header)
		}
		handler.ServeHTTP(rr, req)
	}

	It("Should return 401 when the identity header is missing", func() {
		serve("")
		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 401 when the identity header is not valid base64", func() {
		serve("not-valid-base64!!!")
		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 401 when the identity header is not valid JSON", func() {
		serve(encodeIdentity("{invalid"))
		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 403 for a customer User identity", func() {
		serve(encodeIdentity(`{"identity":{"type":"User","org_id":"000001","user":{"username":"customer"}}}`))
		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 403 for a System identity carrying an associate block", func() {
		serve(encodeIdentity(`{"identity":{"type":"System","org_id":"000001","associate":{"Role":["platform-archive-download"]}}}`))
		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 403 for an Associate type without an associate block", func() {
		serve(encodeIdentity(`{"identity":{"type":"Associate"}}`))
		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 403 for an X509 type without an x509 block", func() {
		serve(encodeIdentity(`{"identity":{"type":"X509","auth_type":"X509"}}`))
		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should return 403 for an X509 identity with an empty subject_dn", func() {
		serve(encodeIdentity(`{"identity":{"type":"X509","auth_type":"X509","x509":{"subject_dn":"","issuer_dn":"/CN=ca"}}}`))
		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())
	})

	It("Should call the next handler for a Turnpike X509 identity", func() {
		serve(encodeIdentity(`{"identity":{"type":"X509","auth_type":"X509","x509":{"subject_dn":"/O=Red Hat/CN=insights-qa","issuer_dn":"/CN=Red Hat IT CA"}}}`))
		Expect(rr.Code).To(Equal(http.StatusOK))
		Expect(nextCalled).To(BeTrue())
		Expect(ctxIdentity.Identity.Type).To(Equal("X509"))
		Expect(ctxIdentity.Identity.X509.SubjectDN).To(Equal("/O=Red Hat/CN=insights-qa"))
	})

	It("Should call the next handler with the identity in context for an Associate identity", func() {
		serve(validIdentityHeader)
		Expect(rr.Code).To(Equal(http.StatusOK))
		Expect(nextCalled).To(BeTrue())
		Expect(ctxIdentity.Identity.Type).To(Equal("Associate"))
		Expect(ctxIdentity.Identity.Associate).ToNot(BeNil())
	})
})

var _ = Describe("Archive link role check", func() {
	It("Should return 403 for a Turnpike X509 identity", func() {
		req, err := test.MakeTestRequest("/api/v1/roles/archiveLink", map[string]interface{}{"page": 0})
		Expect(err).To(BeNil())
		req.Header.Set("x-rh-identity", encodeIdentity(`{"identity":{"type":"X509","auth_type":"X509","x509":{"subject_dn":"/CN=insights-qa","issuer_dn":"/CN=ca"}}}`))
		rr := httptest.NewRecorder()
		http.HandlerFunc(endpoints.RolesArchiveLink).ServeHTTP(rr, req)
		Expect(rr.Code).To(Equal(http.StatusForbidden))
	})

	It("Should return 403 for a non-Associate identity that claims the archive role", func() {
		req, err := test.MakeTestRequest("/api/v1/roles/archiveLink", map[string]interface{}{"page": 0})
		Expect(err).To(BeNil())
		req.Header.Set("x-rh-identity", encodeIdentity(`{"identity":{"type":"System","org_id":"000001","associate":{"Role":["platform-archive-download"]}}}`))
		rr := httptest.NewRecorder()
		http.HandlerFunc(endpoints.RolesArchiveLink).ServeHTTP(rr, req)
		Expect(rr.Code).To(Equal(http.StatusForbidden))
	})
})
