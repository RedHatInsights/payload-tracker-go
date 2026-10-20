package endpoints_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

func getUUID() string {
	newUUID := uuid.New()
	return newUUID.String()
}

func dataPerVerbosity(requestId string, verbosity string, d1 time.Time) structs.SinglePayloadData {
	switch verbosity {
	case "2":
		return structs.SinglePayloadData{
			Service: "puptoo",
			Status:  "recieved",
			Date:    d1,
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
			ID:          1,
			Service:     "puptoo",
			Account:     "test",
			OrgID:       "123456",
			RequestID:   requestId,
			InventoryID: getUUID(),
			SystemID:    getUUID(),
			CreatedAt:   time.Now().Round(0),
			Status:      "received",
			StatusMsg:   "generating reports",
			Date:        d1,
		}
	}
}

func getFourReqIdStatuses(requestId string, verbosity string) []structs.SinglePayloadData {
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

	p5 := p1
	p5.Status = "received"
	p5.Date, _ = time.Parse(time.RFC3339, "2021-08-04T07:45:39.374+00:00")
	p5.Source = "engine"

	p6 := p1
	p6.Status = "success"
	p6.Date, _ = time.Parse(time.RFC3339, "2021-08-04T07:45:39.975+00:00")
	p6.Source = "engine"

	return []structs.SinglePayloadData{p1, p2, p3, p4, p5, p6}
}

var (
	payloadReturnCount int64
	payloadReturnData  []models.Payloads

	reqIdPayloadData []structs.SinglePayloadData
)

func mockedRetrievePayloads(_ *gorm.DB, _ int, _ int, _ structs.Query) (int64, []models.Payloads) {
	return payloadReturnCount, payloadReturnData
}

func mockedRequestIdPayloads(_ *gorm.DB, _ string, _ string, _ string, _ string) []structs.SinglePayloadData {
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
				req, err := test.MakeTestRequest("/api/v1/payloads", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With valid data from DB", func() {
			It("should not mutate any data", func() {
				req, err := test.MakeTestRequest("/api/v1/payloads", query)
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
				req, err := test.MakeTestRequest("/api/v1/payloads", query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_by parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_by"] = "request_id"
				req, err := test.MakeTestRequest("/api/v1/payloads", query)
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
					req, err := test.MakeTestRequest("/api/v1/payloads", query)
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
					req, err := test.MakeTestRequest("/api/v1/payloads", query)
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
		reqIdStatuses := getFourReqIdStatuses(requestId, "2")
		Context("with a valid request", func() {
			It("should return HTTP 200", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdStatuses
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("with an invalid request id, and db returns empty set", func() {
			It("should return HTTP 404", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = make([]structs.SinglePayloadData, 0)
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(404))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.ErrorResponse

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Status).To(Equal(http.StatusNotFound))
			})
		})

		Context("with an invalid request id, and db returns nil", func() {
			It("should return HTTP 404", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = nil
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(404))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.ErrorResponse

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Status).To(Equal(http.StatusNotFound))
			})
		})

		Context("With invalid sort_dir parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_dir"] = "des"
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		Context("With invalid sort_by parameter", func() {
			It("should return HTTP 400", func() {
				query["sort_by"] = "account"
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(400))
				Expect(rr.Body).ToNot(BeNil())
			})
		})

		reqIdStatuses = getFourReqIdStatuses(requestId, "2")
		Context("With valid data from DB", func() {
			It("should pass the data forward", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)

				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdStatuses
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].ID).To(Equal(reqIdStatuses[0].ID))
				Expect(respData.Data[0].Service).To(Equal(reqIdStatuses[0].Service))
				Expect(respData.Data[0].Account).To(Equal(reqIdStatuses[0].Account))
				Expect(respData.Data[0].OrgID).To(Equal(reqIdStatuses[0].OrgID))
				Expect(respData.Data[0].RequestID).To(Equal(reqIdStatuses[0].RequestID))
				Expect(respData.Data[0].InventoryID).To(Equal(reqIdStatuses[0].InventoryID))
				Expect(respData.Data[0].SystemID).To(Equal(reqIdStatuses[0].SystemID))
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(reqIdStatuses[0].CreatedAt.String()))
				Expect(respData.Data[0].Status).To(Equal(reqIdStatuses[0].Status))
				Expect(respData.Data[0].StatusMsg).To(Equal(reqIdStatuses[0].StatusMsg))
				Expect(respData.Data[0].Date.String()).To(Equal(reqIdStatuses[0].Date.String()))
				Expect(respData.Data[1].Source).To(Equal(reqIdStatuses[1].Source))
			})

			It("should correctly calculate durations", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdStatuses
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

		reqIdStatuses = getFourReqIdStatuses(requestId, "1")
		Context("Get to /payloads/{request_id} Verbosity 1", func() {
			It("should pass the data forward", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdStatuses
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].Service).To(Equal(reqIdStatuses[0].Service))
				Expect(respData.Data[0].InventoryID).To(Equal(reqIdStatuses[0].InventoryID))
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(reqIdStatuses[0].CreatedAt.String()))
				Expect(respData.Data[0].Status).To(Equal(reqIdStatuses[0].Status))
				Expect(respData.Data[0].StatusMsg).To(Equal(reqIdStatuses[0].StatusMsg))
				Expect(respData.Data[0].Date.String()).To(Equal(reqIdStatuses[0].Date.String()))
				Expect(respData.Data[1].Date.String()).To(Equal(reqIdStatuses[1].Date.String()))
				Expect(respData.Data[1].Source).To(Equal(reqIdStatuses[1].Source))
			})
		})

		reqIdStatuses = getFourReqIdStatuses(requestId, "0")
		Context("Get to /payloads/{request_id} Verbosity 0", func() {
			It("should pass the data forward", func() {
				req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
				Expect(err).To(BeNil())

				reqIdPayloadData = reqIdStatuses
				handler.ServeHTTP(rr, req)
				Expect(rr.Code).To(Equal(200))
				Expect(rr.Body).ToNot(BeNil())

				var respData structs.PayloadRetrievebyID

				readBody, _ := ioutil.ReadAll(rr.Body)
				json.Unmarshal(readBody, &respData)

				Expect(respData.Data[0].ID).To(Equal(reqIdStatuses[0].ID))
				Expect(respData.Data[0].Service).To(Equal(reqIdStatuses[0].Service))
				Expect(respData.Data[0].Account).To(Equal(reqIdStatuses[0].Account))
				Expect(respData.Data[0].OrgID).To(Equal(reqIdStatuses[0].OrgID))
				Expect(respData.Data[0].RequestID).To(Equal(reqIdStatuses[0].RequestID))
				Expect(respData.Data[0].InventoryID).To(Equal(reqIdStatuses[0].InventoryID))
				Expect(respData.Data[0].SystemID).To(Equal(reqIdStatuses[0].SystemID))
				Expect(respData.Data[0].CreatedAt.String()).To(Equal(reqIdStatuses[0].CreatedAt.String()))
				Expect(respData.Data[0].Status).To(Equal(reqIdStatuses[0].Status))
				Expect(respData.Data[0].StatusMsg).To(Equal(reqIdStatuses[0].StatusMsg))
				Expect(respData.Data[0].Date.String()).To(Equal(reqIdStatuses[0].Date.String()))
				Expect(respData.Data[1].Date.String()).To(Equal(reqIdStatuses[1].Date.String()))
				Expect(respData.Data[1].Source).To(Equal(reqIdStatuses[1].Source))
			})
		})
	})
})

var _ = Describe("PayloadArchiveLink", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder

		requestId string
		query     map[string]interface{}
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()

		// Mock out the storage broker server.  This allows us to test the response handling code.
		mockStorageBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{\"url\": \"www.example.com\"}"))
		}))

		handler = http.HandlerFunc(endpoints.PayloadArchiveLink(endpoints.RequestArchiveLink(mockStorageBrokerServer.URL, 10)))

		requestId = getUUID()
		query = make(map[string]interface{})
	})

	Context("When the request_id is not a valid UUID", func() {
		It("Should return 400", func() {
			req, err := test.MakeTestRequest("/api/v1/payloads/1234/archiveLink", query)
			Expect(err).To(BeNil())
			req.Header.Set("x-rh-identity", validIdentityHeader)
			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("With a missing Identity header", func() {
		It("Should return 401", func() {
			req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s/archiveLink", requestId), query)
			Expect(err).To(BeNil())
			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("Without the required role", func() {
		It("Should return 403", func() {
			req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s/archiveLink", requestId), query)
			Expect(err).To(BeNil())
			req.Header.Set("x-rh-identity", invalidIdentityHeader)
			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})
	})

	Context("With a valid request_id and the required roles in the Identity header", func() {
		It("Should return the payload archive's URL", func() {
			req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s/archiveLink", requestId), query)
			Expect(err).To(BeNil())
			req.Header.Set("x-rh-identity", validIdentityHeader)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("request_id", requestId)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body).ToNot(BeNil())

			var respData structs.PayloadArchiveLink

			readBody, _ := ioutil.ReadAll(rr.Body)
			json.Unmarshal(readBody, &respData)

			Expect(respData.Url).To(Equal("www.example.com"))
		})
	})

})

var _ = Describe("PayloadKibanaLink", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder

		requestId string
		cfg       *config.TrackerConfig
		query     map[string]interface{}
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(endpoints.PayloadKibanaLink)

		requestId = getUUID()
		cfg = config.Get()
		query = make(map[string]interface{})
	})

	Context("When the request_id is not a valid UUID", func() {
		It("Should return 400", func() {
			req, err := test.MakeTestRequest("/api/v1/payloads/1234/kibanaLink", query)
			Expect(err).To(BeNil())
			req.Header.Set("x-rh-identity", validIdentityHeader)
			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("With a valid request_id", func() {
		It("Should return a Kibana dashboard URL", func() {
			fmt.Println("Request URL:", fmt.Sprintf("/api/v1/payloads/%s/kibanaLink", requestId))
			req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s/kibanaLink", requestId), query)
			Expect(err).To(BeNil())
			req.Header.Set("x-rh-identity", validIdentityHeader)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("request_id", requestId)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body).ToNot(BeNil())

			var respData structs.PayloadKibanaLink
			readBody, _ := ioutil.ReadAll(rr.Body)
			json.Unmarshal(readBody, &respData)

			Expect(respData.Url).To(Not(BeNil()))
			Expect(respData.Url).To(ContainSubstring(cfg.KibanaConfig.DashboardURL))
			Expect(respData.Url).To(ContainSubstring(cfg.KibanaConfig.Index))
			Expect(respData.Url).To(ContainSubstring(requestId))
		})

		It("Should filter by service", func() {
			query["service"] = "testService"
			req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s/kibanaLink", requestId), query)
			Expect(err).To(BeNil())
			req.Header.Set("x-rh-identity", validIdentityHeader)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("request_id", requestId)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body).ToNot(BeNil())

			var respData structs.PayloadKibanaLink
			readBody, _ := ioutil.ReadAll(rr.Body)
			json.Unmarshal(readBody, &respData)

			Expect(respData.Url).To(Not(BeNil()))
			Expect(respData.Url).To(ContainSubstring(cfg.KibanaConfig.DashboardURL))
			Expect(respData.Url).To(ContainSubstring(cfg.KibanaConfig.Index))
			Expect(respData.Url).To(ContainSubstring(requestId))
			Expect(respData.Url).To(ContainSubstring("testService"))
		})
	})
})
