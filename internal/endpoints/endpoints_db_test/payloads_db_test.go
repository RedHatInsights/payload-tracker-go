package endpoints_db_test

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

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

var _ = Describe("Payloads with DB", func() {
	var (
		handler http.Handler
		rr      *httptest.ResponseRecorder
		query   map[string]interface{}
	)

	db := test.WithDatabase()

	BeforeEach(func() {
		rr = httptest.NewRecorder()

		query = make(map[string]interface{})

		endpoints.Db = db
	})

	Context("With payloads data in DB", func() {
		It("retrieves payload", func() {
			handler = http.HandlerFunc(endpoints.Payloads)

			inventoryId := uuid.New().String()

			query["inventory_id"] = inventoryId
			req, err := test.MakeTestRequest("/api/v1/payloads", query)
			Expect(err).To(BeNil())

			payloadData := models.Payloads{
				Account:     "test",
				RequestId:   uuid.New().String(),
				InventoryId: inventoryId,
				SystemId:    uuid.New().String(),
			}

			Expect(db().Create(&payloadData).Error).ToNot(HaveOccurred())

			handler.ServeHTTP(rr, req)
			Expect(rr.Code).To(Equal(200))
			Expect(rr.Body).ToNot(BeNil())

			payloadRespData := structs.PayloadsData{}

			readBody, _ := ioutil.ReadAll(rr.Body)
			json.Unmarshal(readBody, &payloadRespData)

			Expect(payloadRespData.Data[0].RequestId).To(Equal(payloadData.RequestId))
			Expect(payloadRespData.Data[0].InventoryId).To(Equal(payloadData.InventoryId))
			Expect(payloadRespData.Data[0].SystemId).To(Equal(payloadData.SystemId))
		})
	})

	Context("With payload statuses data in DB", func() {
		It("retrieves request_id payload", func() {
			handler = http.HandlerFunc(endpoints.RequestIdPayloads)

			requestId := uuid.New().String()

			payloadData := models.Payloads{
				Account:     "test",
				RequestId:   requestId,
				InventoryId: uuid.New().String(),
				SystemId:    uuid.New().String(),
			}
			statusData := models.Statuses{Name: "test-status"}
			sourceData := models.Sources{Name: "test-source"}
			serviceData := models.Services{Name: "test-service"}

			Expect(db().Create(&statusData).Error).ToNot(HaveOccurred())
			Expect(db().Create(&sourceData).Error).ToNot(HaveOccurred())
			Expect(db().Create(&serviceData).Error).ToNot(HaveOccurred())
			Expect(db().Create(&payloadData).Error).ToNot(HaveOccurred())

			payloadDate, _ := time.Parse(time.RFC3339, "2022-06-03T14:00:32.253Z")
			payloadStatusData := models.PayloadStatuses{
				PayloadId: payloadData.Id,
				Status:    statusData,
				Source:    sourceData,
				Service:   serviceData,
				StatusMsg: "test-status-msg",
				Date:      payloadDate,
			}
			Expect(db().Create(&payloadStatusData).Error).ToNot(HaveOccurred())

			req, err := test.MakeTestRequest(fmt.Sprintf("/api/v1/payloads/%s", requestId), query)
			Expect(err).To(BeNil())

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("request_id", requestId)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(200))
			Expect(rr.Body).ToNot(BeNil())

			respData := structs.PayloadRetrievebyID{}

			readBody, _ := ioutil.ReadAll(rr.Body)
			json.Unmarshal(readBody, &respData)

			Expect(respData.Data[0].Service).To(Equal(payloadStatusData.Service.Name))
			Expect(respData.Data[0].Account).To(Equal(payloadData.Account))
			Expect(respData.Data[0].OrgID).To(Equal(payloadData.OrgId))
			Expect(respData.Data[0].RequestID).To(Equal(payloadData.RequestId))
			Expect(respData.Data[0].InventoryID).To(Equal(payloadData.InventoryId))
			Expect(respData.Data[0].SystemID).To(Equal(payloadData.SystemId))
			Expect(respData.Data[0].Status).To(Equal(payloadStatusData.Status.Name))
			Expect(respData.Data[0].StatusMsg).To(Equal(payloadStatusData.StatusMsg))
			Expect(respData.Data[0].Source).To(Equal(payloadStatusData.Source.Name))
		})
	})
})
