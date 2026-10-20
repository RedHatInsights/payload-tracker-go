package endpoints_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

const validIdentityHeader = "eyJpZGVudGl0eSI6IHsiYXNzb2NpYXRlIjp7IlJvbGUiOlsicGxhdGZvcm0tYXJjaGl2ZS1kb3dubG9hZCIsIm90aGVyUm9sZSJdfSwgImFjY291bnRfbnVtYmVyIjogIjAwMDAwMDEiLCAidHlwZSI6ICJTeXN0ZW0iLCAiaW50ZXJuYWwiOiB7Im9yZ19pZCI6ICIwMDAwMDEifX19"
const invalidIdentityHeader = "eyJpZGVudGl0eSI6IHsiYXNzb2NpYXRlIjp7IlJvbGUiOlsib3RoZXJSb2xlIl19LCAiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJ0eXBlIjogIlN5c3RlbSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0="

var _ = Describe("Roles", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder
		query   map[string]interface{}
	)

	Describe("Get the archiveLink role endpoint", func() {

		Context("With a missing Identity header", func() {
			It("Should return 401", func() {
				req, err := test.MakeTestRequest("/api/v1/roles/archiveLink", query)
				Expect(err).To(BeNil())
				handler = http.HandlerFunc(endpoints.RolesArchiveLink)
				rr = httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("Without the required role", func() {
			It("Should return 403", func() {
				req, err := test.MakeTestRequest("/api/v1/roles/archiveLink", query)
				Expect(err).To(BeNil())
				req.Header.Set("x-rh-identity", invalidIdentityHeader)
				handler = http.HandlerFunc(endpoints.RolesArchiveLink)
				rr = httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(http.StatusForbidden))
			})
		})

		Context("With a valid Identity header containing the required role", func() {
			It("Should return 200", func() {
				req, err := test.MakeTestRequest("/api/v1/roles/archiveLink", query)
				Expect(err).To(BeNil())
				req.Header.Set("x-rh-identity", validIdentityHeader)
				handler = http.HandlerFunc(endpoints.RolesArchiveLink)
				rr = httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(http.StatusOK))
			})
		})

	})
})
