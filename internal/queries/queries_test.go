package queries

import (
	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testDBImpl struct {
	DB               PayloadFieldsRepositoryFromDB
	GetStatusCalled  bool
	GetServiceCalled bool
	GetSourceCalled  bool
}

func (d *testDBImpl) GetStatus(statusName string) models.Statuses {
	d.GetStatusCalled = true

	return models.Statuses{Id: 1234, Name: statusName}
}

func (d *testDBImpl) GetService(serviceName string) models.Services {
	d.GetServiceCalled = true

	return models.Services{Id: 1234, Name: serviceName}
}

func (d *testDBImpl) GetSource(sourceName string) models.Sources {
	d.GetSourceCalled = true

	return models.Sources{Id: 1234, Name: sourceName}
}

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

		result, id := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())
		Expect(id).ToNot(Equal(uint(0)))

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

		result, id := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())
		Expect(id).ToNot(Equal(uint(0)))

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

		result, id := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())
		Expect(id).ToNot(Equal(uint(0)))

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

		result, id := UpsertPayloadByRequestId(db(), requestId, payload)
		Expect(result.Error).ToNot(HaveOccurred())
		Expect(id).ToNot(Equal(uint(0)))

		payload, err := GetPayloadByRequestId(db(), requestId)
		Expect(err).ToNot(HaveOccurred())

		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.InventoryId).To(Equal(inventoryId))
		Expect(payload.SystemId).To(Equal(systemId))
		Expect(payload.Account).To(Equal("1234"))
		Expect(payload.OrgId).To(Equal("1234"))
	})
	It("Checks if we got a cached status result from the database", func() {
		const statusName string = "TestStatus"
		dbImpl := testDBImpl{}
		cachedDBImpl := PayloadFieldsRepositoryFromCache{DB: &dbImpl, StatusCache: make(map[string]models.Statuses), ServiceCache: make(map[string]models.Services), SourceCache: make(map[string]models.Sources)}

		// Cache miss
		cachedDBImpl.GetStatus(statusName)

		Expect(dbImpl.GetStatusCalled).To(Equal(true))

		// Cache hit
		dbImpl.GetStatusCalled = false
		cachedDBImpl.GetStatus(statusName)

		Expect(dbImpl.GetStatusCalled).To(Equal(false))
	})
	It("Checks if we got a cached service result from the database", func() {
		const serviceName = "TestService"

		dbImpl := testDBImpl{}
		cachedDBImpl := PayloadFieldsRepositoryFromCache{DB: &dbImpl, StatusCache: make(map[string]models.Statuses), ServiceCache: make(map[string]models.Services), SourceCache: make(map[string]models.Sources)}

		// Cache miss
		cachedDBImpl.GetStatus(serviceName)

		Expect(dbImpl.GetStatusCalled).To(Equal(true))

		// Cache hit
		dbImpl.GetStatusCalled = false
		cachedDBImpl.GetStatus(serviceName)

		Expect(dbImpl.GetStatusCalled).To(Equal(false))
	})
	It("Checks if we got a cached source result from the database", func() {
		const sourceName = "TestSource"
		dbImpl := testDBImpl{}
		cachedDBImpl := PayloadFieldsRepositoryFromCache{DB: &dbImpl, StatusCache: make(map[string]models.Statuses), ServiceCache: make(map[string]models.Services), SourceCache: make(map[string]models.Sources)}

		// Cache miss
		cachedDBImpl.GetStatus(sourceName)

		Expect(dbImpl.GetStatusCalled).To(Equal(true))

		// Cache hit
		dbImpl.GetStatusCalled = false
		cachedDBImpl.GetStatus(sourceName)

		Expect(dbImpl.GetStatusCalled).To(Equal(false))
	})
})
