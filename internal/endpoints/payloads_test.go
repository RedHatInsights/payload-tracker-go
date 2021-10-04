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

func dataPerVerbosity(requestId string, verbosity string, d1 time.Time) structs.SinglePayloadData {
	switch verbosity {
	case "2":
		return structs.SinglePayloadData{
			ID:          1,
			Service:     "puptoo",
			Account:     "test",
			RequestID:   requestId,
			InventoryID: getUUID(),
			SystemID:    getUUID(),
			CreatedAt:   time.Now().Round(0),
			Status:      "received",
			StatusMsg:   "generating reports",
			Date:        d1,
		}
	case "1":
		return structs.SinglePayloadData{
			Service:     "puptoo",
			Status:      "recieved",
			InventoryID: getUUID(),
			Date:        d1,
			StatusMsg:   "generating reports",
		}
	default:
		return structs.SinglePayloadData{
			Service: "puptoo",
			Status:  "recieved",
			Date:    d1,
		}
	}
}

func getFourReqIdPayloads(requestId string, verbosity string) []structs.SinglePayloadData {
	d1, _ := time.Parse(time.RFC3339, "2021-08-04T07:45:26.371Z")
	p1 := dataPerVerbosity(requestId, verbosity, d1)

	p2 := p1
	p2.Status = "processing"
	p2.Date, _ = time.Parse(time.RFC3339, "2021-08-04T07:45:33.350Z")
	p2.Source = "inventory"

	p3 := p1
	p3.Status = "processed"
	p3.Date, _ = time.Parse(time.RFC3339, "2021-08-04T07:45:36.341+00:00")

	p4 := p1
	p4.Status = "success"
	p4.Date, _ = time.Parse(time.RFC3339, "2021-08-04T07:45:38.975+00:00")
	p4.Source = "inventory"

	return []structs.SinglePayloadData{p1, p2, p3, p4}
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

	reqIdPayloadData []structs.SinglePayloadData
)

func mockedRetrievePayloads(_ int, _ int, _ structs.Query) (int64, []models.Payloads) {
	return payloadReturnCount, payloadReturnData
}

func mockedRequestIdPayloads(_ string, _ string, _ string, _ string) []structs.SinglePayloadData {
	return reqIdPayloadData
}

var _ = Describe("Payloads", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder
		query   map[string]interface{}
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(endpoints.Payloads)

		endpoints.RetrievePayloads = mockedRetrievePayloads
		query = make(map[string]interface{})
	})

	Describe("Get to payloads endpoint", func() {
		Context("With a valid request", func() {
			It("should return HTTP 200", func() {
				req, err := makeTestRequest("/api/v1/payloads", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With valid data from DB", func() {
			It("should not mutate any data", func() {
				req, err := makeTestRequest("/api/v1/payloads", query)
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
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(payloadData.CreatedAt.String()))
			})
		})

		Context("With invalid sort_dir parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_dir"] = "ascs"
				req, err := makeTestRequest("/api/v1/payloads", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_by parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_by"] = "request_id"
				req, err := makeTestRequest("/api/v1/payloads", query)
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
					query = make(map[string]interface{})
					query[k] = v
					req, err := makeTestRequest("/api/v1/payloads", query)
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
					query = make(map[string]interface{})
					query[k] = v
					req, err := makeTestRequest("/api/v1/payloads", query)
					Expect(err).To(BeNil())
					handler.ServeHTTP(rr, req)
					Expect(rr.Code).To(Equal(400))
					Expect(rr.Body).ToNot(BeNil())
				}
			})
		})
	})

})

var _ = Describe("RequestIdPayloads", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder

		requestId string
		query     map[string]interface{}
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(endpoints.RequestIdPayloads)

		endpoints.RetrieveRequestIdPayloads = mockedRequestIdPayloads
		requestId = getUUID()
		query = make(map[string]interface{})
	})

	Describe("Get to /payloads/{request_id}", func() {
		Context("with a valid request", func() {
			It("should return HTTP 200", func() {
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_dir parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_dir"] = "des"
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_by parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_by"] = "account"
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		reqIdPayloads := getFourReqIdPayloads(requestId, "2")
		Context("With valid data from DB", func() {
			It("should pass the data forward", func() {
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdPayloads
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].ID).To(Equal(reqIdPayloads[0].ID))
				Expect(respData.Data[0].Service).To(Equal(reqIdPayloads[0].Service))
				Expect(respData.Data[0].Account).To(Equal(reqIdPayloads[0].Account))
				Expect(respData.Data[0].RequestID).To(Equal(reqIdPayloads[0].RequestID))
				Expect(respData.Data[0].InventoryID).To(Equal(reqIdPayloads[0].InventoryID))
				Expect(respData.Data[0].SystemID).To(Equal(reqIdPayloads[0].SystemID))
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(reqIdPayloads[0].CreatedAt.String()))
				Expect(respData.Data[0].Status).To(Equal(reqIdPayloads[0].Status))
				Expect(respData.Data[0].StatusMsg).To(Equal(reqIdPayloads[0].StatusMsg))
				Expect(respData.Data[0].Date.String()).To(Equal(reqIdPayloads[0].Date.String()))
			})

			It("should correctly calculate durations", func() {
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdPayloads
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Durations["puptoo:inventory"]).To(Equal("00:00:05.625000"))
				Expect(respData.Durations["puptoo:undefined"]).To(Equal("00:00:09.970000"))
			})
		})

		reqIdPayloads = getFourReqIdPayloads(requestId, "1")
		Context("Get to /payloads/{request_id} Verbosity 1", func() {
			It("should pass the data forward", func() {
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdPayloads
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].SystemID).To(BeEmpty())
				Expect(respData.Data[0].RequestID).To(BeEmpty())
				Expect(respData.Data[0].Service).To(Equal(reqIdPayloads[0].Service))
				Expect(respData.Data[0].InventoryID).To(Equal(reqIdPayloads[0].InventoryID))
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(reqIdPayloads[0].CreatedAt.String()))
				Expect(respData.Data[0].Status).To(Equal(reqIdPayloads[0].Status))
				Expect(respData.Data[0].StatusMsg).To(Equal(reqIdPayloads[0].StatusMsg))
				Expect(respData.Data[0].Date.String()).To(Equal(reqIdPayloads[0].Date.String()))
			})
		})

		reqIdPayloads = getFourReqIdPayloads(requestId, "0")
		Context("Get to /payloads/{request_id} Verbosity 0", func() {
			It("should pass the data forward", func() {
				req, err := makeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdPayloads
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].SystemID).To(BeEmpty())
				Expect(respData.Data[0].RequestID).To(BeEmpty())
				Expect(respData.Data[0].StatusMsg).To(BeEmpty())
				Expect(respData.Data[0].Service).To(Equal(reqIdPayloads[0].Service))
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(reqIdPayloads[0].CreatedAt.String()))
				Expect(respData.Data[0].Status).To(Equal(reqIdPayloads[0].Status))
				Expect(respData.Data[0].Date.String()).To(Equal(reqIdPayloads[0].Date.String()))
			})
		})
	})
})
