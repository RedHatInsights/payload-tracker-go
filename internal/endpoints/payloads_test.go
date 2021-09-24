package endpoints_test

import (
	"fmt"
	"net/http/httptest"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

func formattedQuery(params map[string]interface{}) string {
	formatted := ""
	for k,v := range(params) {
		formatted += fmt.Sprintf("&%v=%v",k,v)
	}
	return formatted[1:]
}


func makeTestRequest(uri string, queryParams map[string]interface{}) (*http.Request, error) {
	var req *http.Request
	var err error

	fullURI := uri
	if len(queryParams) > 0 {
		fullURI += "?"+formattedQuery(queryParams)
	}

	req, err = http.NewRequest("GET", fullURI, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

var (
	payloadReturnCount int64
	payloadReturnData []models.Payloads
)

func mockedRetrievePayloads(_ int, _ int, _ structs.Query) (int64, []models.Payloads) {
	return payloadReturnCount, payloadReturnData
}


var _ = Describe("Payloads", func() {
	var (
		handler http.Handler
		rr *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(endpoints.Payloads)

		endpoints.RetrievePayloads = mockedRetrievePayloads
	})

	Describe("Get to payloads endpoint", func() {
		Context("With valid data from db", func() {
			It("should return HTTP 200", func() {
				query := make(map[string]interface{})
				req, err := makeTestRequest("/api/payloads/v1", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})
	})

})
