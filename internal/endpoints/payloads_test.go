package endpoints_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

func getUUID() string {
	newUUID := uuid.New()
	return newUUID.String()
}

func formattedQuery(params map[string]interface{}) string {
	formatted := ""
	for k, v := range params {
		formatted += fmt.Sprintf("&%v=%v", k, v)
	}
	return formatted[1:]
}

func makeTestRequest(uri string, queryParams map[string]interface{}) (*http.Request, error) {
	var req *http.Request
	var err error

	fullURI := uri
	if len(queryParams) > 0 {
		fullURI += "?" + formattedQuery(queryParams)
	}

	req, err = http.NewRequest("GET", fullURI, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

var (
	payloadReturnCount int64
	payloadReturnData  []models.Payloads
)

func mockedRetrievePayloads(_ int, _ int, _ structs.Query) (int64, []models.Payloads) {
	return payloadReturnCount, payloadReturnData
}

var _ = Describe("Payloads", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(endpoints.Payloads)

		endpoints.RetrievePayloads = mockedRetrievePayloads
	})

	Describe("Get to payloads endpoint", func() {
		Context("With a valid request", func() {
			It("should return HTTP 200", func() {
				query := make(map[string]interface{})
				req, err := makeTestRequest("/api/payloads/v1", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With valid data from DB", func() {
			It("should not mutate any data", func() {
				query := make(map[string]interface{})
				req, err := makeTestRequest("/api/payloads/v1", query)
				Expect(err).To(BeNil())

				payloadData := models.Payloads{
					Id:          1,
					RequestId:   getUUID(),
					InventoryId: getUUID(),
					SystemId:    getUUID(),
					CreatedAt:   time.Now().Round(0),
				}

				payloadReturnCount = 1
				payloadReturnData = []models.Payloads{payloadData}

				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadsData

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].Id).To(Equal(payloadData.Id))
				Expect(respData.Data[0].RequestId).To(Equal(payloadData.RequestId))
				Expect(respData.Data[0].InventoryId).To(Equal(payloadData.InventoryId))
				Expect(respData.Data[0].SystemId).To(Equal(payloadData.SystemId))
				Expect(respData.Data[0].CreatedAt).To(Equal(payloadData.CreatedAt))
			})
		})

		Context("With invalid sort_dir parameter", func() {
			It("should return HTTP 400", func() {
				query := make(map[string]interface{})
				query["sort_dir"] = "ascs"
				req, err := makeTestRequest("/api/payloads/v1", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_by parameter", func() {
			It("should return HTTP 400", func() {
				query := make(map[string]interface{})
				query["sort_by"] = "request_id"
				req, err := makeTestRequest("/api/payloads/v1", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		validTimestamps := map[string]string{
			"created_at_lt":  "2021-08-04T17:53:29.724476-04:00",
			"created_at_lte": "2021-08-04T17:53:29.724476-04:00",
			"created_at_gt":  "2021-08-04T17:46:22.078999-04:00",
			"created_at_gte": "2021-08-04T17:46:22.078999-04:00",
		}
		Context("With valid timestamps query parameter", func() {
			It("should return HTTP 200", func() {
				for k, v := range validTimestamps {
					query := make(map[string]interface{})
					query[k] = v
					req, err := makeTestRequest("/api/payloads/v1", query)
					Expect(err).To(BeNil())
					handler.ServeHTTP(rr, req)
					Expect(rr.Code).To(Equal(200))
					Expect(rr.Body).ToNot(BeNil())
				}
			})
		})

		invalidTimestamps := map[string]string{
			"created_at_lt":  "invalid",
			"created_at_lte": "nope",
			"created_at_gt":  "nah",
			"created_at_gte": "nice try..but no",
		}
		Context("With invalid timestamps query parameter", func() {
			It("should return HTTP 400", func() {
				for k, v := range invalidTimestamps {
					query := make(map[string]interface{})
					query[k] = v
					req, err := makeTestRequest("/api/payloads/v1", query)
					Expect(err).To(BeNil())
					handler.ServeHTTP(rr, req)
					Expect(rr.Code).To(Equal(400))
					Expect(rr.Body).ToNot(BeNil())
				}
			})
		})
	})

})
