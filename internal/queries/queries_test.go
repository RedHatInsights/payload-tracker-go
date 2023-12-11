package queries

import (
	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func getUUID() string {
	return uuid.New().String()
}

var _ = Describe("Queries", func() {
	db := test.WithDatabase()

	It("Retrieves request id payload", func() {
		requestId := getUUID()
		payload := models.Payloads{
			RequestId:   requestId,
			Account:     "1234",
			OrgId:       "1234",
			InventoryId: getUUID(),
			SystemId:    getUUID(),
		}
		Expect(db().Create(&payload).Error).ToNot(HaveOccurred())

		payload, err := GetPayloadByRequestId(db(), requestId)

		Expect(err).ToNot(HaveOccurred())

		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.Account).To(Equal("1234"))
		Expect(payload.OrgId).To(Equal("1234"))
	})
	It("Updates payload for request id", func() {
		requestId := getUUID()
		inventoryId := getUUID()
		systemId := getUUID()
		payload := models.Payloads{
			RequestId:   requestId,
			Account:     "1234",
			OrgId:       "1234",
			InventoryId: inventoryId,
			SystemId:    systemId,
		}
		Expect(db().Create(&payload).Error).ToNot(HaveOccurred())

		payload = models.Payloads{
			RequestId:   requestId,
			Account:     "5678",
			OrgId:       "5678",
			InventoryId: inventoryId,
			SystemId:    systemId,
		}

		result, _ := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())

		payload, err := GetPayloadByRequestId(db(), requestId)
		Expect(err).ToNot(HaveOccurred())

		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.InventoryId).To(Equal(inventoryId))
		Expect(payload.Account).To(Equal("5678"))
		Expect(payload.OrgId).To(Equal("5678"))
	})
	It("Updates without storing empty account/org_id for request id", func() {
		requestId := getUUID()
		inventoryId := getUUID()
		systemId := getUUID()
		payload := models.Payloads{
			RequestId:   requestId,
			Account:     "1234",
			OrgId:       "1234",
			InventoryId: inventoryId,
			SystemId:    systemId,
		}
		Expect(db().Create(&payload).Error).ToNot(HaveOccurred())

		payload = models.Payloads{
			RequestId:   requestId,
			InventoryId: getUUID(),
			SystemId:    getUUID(),
		}

		result, _ := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())

		payload, err := GetPayloadByRequestId(db(), requestId)
		Expect(err).ToNot(HaveOccurred())

		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.InventoryId).ToNot(Equal(inventoryId))
		Expect(payload.SystemId).ToNot(Equal(systemId))
		Expect(payload.Account).To(Equal("1234"))
		Expect(payload.OrgId).To(Equal("1234"))
	})
	It("Updates without storing empty inventory_id/system_id for request id", func() {
		requestId := getUUID()
		inventoryId := getUUID()
		systemId := getUUID()
		payload := models.Payloads{
			RequestId:   requestId,
			Account:     "1234",
			OrgId:       "1234",
			InventoryId: inventoryId,
			SystemId:    systemId,
		}
		Expect(db().Create(&payload).Error).ToNot(HaveOccurred())

		payload = models.Payloads{
			RequestId: requestId,
			Account:   "5678",
			OrgId:     "5678",
		}

		result, _ := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())

		payload, err := GetPayloadByRequestId(db(), requestId)
		Expect(err).ToNot(HaveOccurred())

		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.InventoryId).To(Equal(inventoryId))
		Expect(payload.SystemId).To(Equal(systemId))
		Expect(payload.Account).To(Equal("5678"))
		Expect(payload.OrgId).To(Equal("5678"))
	})
	It("Updates nothing if all fields are empty", func() {
		requestId := getUUID()
		inventoryId := getUUID()
		systemId := getUUID()
		payload := models.Payloads{
			RequestId:   requestId,
			Account:     "1234",
			OrgId:       "1234",
			InventoryId: inventoryId,
			SystemId:    systemId,
		}
		Expect(db().Create(&payload).Error).ToNot(HaveOccurred())

		payload = models.Payloads{
			RequestId: requestId,
		}

		result, _ := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())

		payload, err := GetPayloadByRequestId(db(), requestId)
		Expect(err).ToNot(HaveOccurred())

		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.InventoryId).To(Equal(inventoryId))
		Expect(payload.SystemId).To(Equal(systemId))
		Expect(payload.Account).To(Equal("1234"))
		Expect(payload.OrgId).To(Equal("1234"))
	})
})
