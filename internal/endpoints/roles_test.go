package endpoints_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

const validIdentityHeader = "eyJpZGVudGl0eSI6eyJ0eXBlIjoiQXNzb2NpYXRlIiwiYXV0aF90eXBlIjoic2FtbC1hdXRoIiwiYXNzb2NpYXRlIjp7IlJvbGUiOlsicGxhdGZvcm0tYXJjaGl2ZS1kb3dubG9hZCIsIm90aGVyUm9sZSJdLCJlbWFpbCI6Impkb2VAcmVkaGF0LmNvbSJ9LCJpbnRlcm5hbCI6eyJvcmdfaWQiOiIwMDAwMDEifX19"
const invalidIdentityHeader = "eyJpZGVudGl0eSI6eyJ0eXBlIjoiQXNzb2NpYXRlIiwiYXV0aF90eXBlIjoic2FtbC1hdXRoIiwiYXNzb2NpYXRlIjp7IlJvbGUiOlsib3RoZXJSb2xlIl0sImVtYWlsIjoiamRvZUByZWRoYXQuY29tIn0sImludGVybmFsIjp7Im9yZ19pZCI6IjAwMDAwMSJ9fX0="

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
