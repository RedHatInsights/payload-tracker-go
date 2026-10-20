package endpoints_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/api"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

var _ = Describe("OpenAPISpec", func() {
	It("Should return 200 with the embedded spec as the body", func() {
		req, err := test.MakeTestRequest("/api/v1/openapi.json", nil)
		Expect(err).To(BeNil())

		rr := httptest.NewRecorder()
		http.HandlerFunc(endpoints.OpenAPISpec).ServeHTTP(rr, req)

		Expect(rr.Code).To(Equal(http.StatusOK))
		Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))
		Expect(rr.Body.String()).To(Equal(string(api.Spec)))
	})
})
