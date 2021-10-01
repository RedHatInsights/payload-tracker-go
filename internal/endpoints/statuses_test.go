package endpoints_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
)

var (
	statusPayloadCount  int64
	statusesPayloadData []structs.StatusRetrieve
)

func mockedRetrieveStatuses(_ structs.Query) (int64, []structs.StatusRetrieve) {
	return statusPayloadCount, statusesPayloadData
}

var _ = Describe("Statuses", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder
		query   map[string]interface{}
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(endpoints.Statuses)

		endpoints.RetrieveStatuses = mockedRetrieveStatuses
		query = make(map[string]interface{})
	})

	Describe("Get to statuses endpoint", func() {
		Context("With a valid request", func() {
			It("Should return 200", func() {
				req, err := makeTestRequest("/api/v1/statuses", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With valid data from DB", func() {
			It("should not mutate any data", func() {
				req, err := makeTestRequest("/api/v1/statuses", query)
				Expect(err).To(BeNil())

				payloadStatusData := structs.StatusRetrieve{
					RequestID: getUUID(),
					Status:    "processed",
					ID:        "1",
					Service:   "puptoo",
					Source:    "inventory",
					StatusMsg: "generating reports",
					Date:      "2021-08-04T07:45:26.371-04:00",
					CreatedAt: "2021-08-04T17:46:22.091375-04:0",
				}

				statusPayloadCount = 1
				statusesPayloadData = []structs.StatusRetrieve{payloadStatusData}

				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.StatusesData

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].ID).To(Equal(payloadStatusData.ID))
				Expect(respData.Data[0].RequestID).To(Equal(payloadStatusData.RequestID))
				Expect(respData.Data[0].Status).To(Equal(payloadStatusData.Status))
				Expect(respData.Data[0].Service).To(Equal(payloadStatusData.Service))
				Expect(respData.Data[0].Source).To(Equal(payloadStatusData.Source))
				Expect(respData.Data[0].StatusMsg).To(Equal(payloadStatusData.StatusMsg))
				Expect(respData.Data[0].Date).To(Equal(payloadStatusData.Date))
				Expect(respData.Data[0].CreatedAt).To(Equal(payloadStatusData.CreatedAt))
			})
		})

		Context("With invalid sort_dir parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_dir"] = "ascs"
				req, err := makeTestRequest("/api/v1/statuses", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_by parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_by"] = "account"
				req, err := makeTestRequest("/api/v1/statuses", query)
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
			"date_lt":        "2021-08-04T17:53:29.724476-04:00",
			"date_lte":       "2021-08-04T17:53:29.724476-04:00",
			"date_gt":        "2021-08-04T17:46:22.078999-04:00",
			"date_gte":       "2021-08-04T17:46:22.078999-04:00",
		}
		Context("With valid timestamps query parameter", func() {
			It("should return HTTP 200", func() {
				for k, v := range validTimestamps {
					query = make(map[string]interface{})
					query[k] = v
					req, err := makeTestRequest("/api/v1/statuses", query)
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
			"date_lt":        "notvalid",
			"date_lte":       "pumpkinspicelatte",
			"date_gt":        "halloween",
			"date_gte":       "trickortreat",
		}
		Context("With invalid timestamps query parameter", func() {
			It("should return HTTP 400", func() {
				for k, v := range invalidTimestamps {
					query = make(map[string]interface{})
					query[k] = v
					req, err := makeTestRequest("/api/v1/statuses", query)
					Expect(err).To(BeNil())
					handler.ServeHTTP(rr, req)
					Expect(rr.Code).To(Equal(400))
					Expect(rr.Body).ToNot(BeNil())
				}
			})
		})
	})
})
