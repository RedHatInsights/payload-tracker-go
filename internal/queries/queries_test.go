package queries

import (
	"fmt"

	models "github.com/redhatinsights/payload-tracker-go/internal/models/db"
	"github.com/redhatinsights/payload-tracker-go/internal/redis"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockPayloadFieldsRepository struct {
	getStatusCalled  bool
	getServiceCalled bool
	getSourceCalled  bool
}

func (m *mockPayloadFieldsRepository) GetStatus(statusName string) models.Statuses {
	m.getStatusCalled = true

	return models.Statuses{Id: 1234, Name: statusName}
}

func (m *mockPayloadFieldsRepository) GetService(serviceName string) models.Services {
	m.getServiceCalled = true

	return models.Services{Id: 1234, Name: serviceName}
}

func (m *mockPayloadFieldsRepository) GetSource(sourceName string) models.Sources {
	m.getSourceCalled = true

	return models.Sources{Id: 1234, Name: sourceName}
}

func getUUID() string {
	return uuid.New().String()
}

func testStatusCaching(payloadFieldsRepository PayloadFieldsRepository, mockPayloadFieldsRepository *mockPayloadFieldsRepository, statusName string) {
	// Cache miss
	payloadReturned := payloadFieldsRepository.GetStatus(statusName)
	// __AUTO_GENERATED_PRINT_VAR_START__
	fmt.Println(fmt.Sprintf("MISS testStatusCaching payloadReturned: %v", payloadReturned)) // __AUTO_GENERATED_PRINT_VAR_END__

	Expect(mockPayloadFieldsRepository.getStatusCalled).To(Equal(true))
	// __AUTO_GENERATED_PRINT_VAR_START__
	fmt.Println(fmt.Sprintf("testStatusCaching mockPayloadFieldsRepository.getStatusCalled: %v", mockPayloadFieldsRepository.getStatusCalled)) // __AUTO_GENERATED_PRINT_VAR_END__
	Expect(payloadReturned.Id).To(Equal(int32(1234)))
	Expect(payloadReturned.Name).To(Equal(statusName))

	// Cache hit
	mockPayloadFieldsRepository.getStatusCalled = false
	// __AUTO_GENERATED_PRINT_VAR_START__
	fmt.Println(fmt.Sprintf("testStatusCaching mockPayloadFieldsRepository.getStatusCalled: %v", mockPayloadFieldsRepository.getStatusCalled)) // __AUTO_GENERATED_PRINT_VAR_END__
	payloadReturned = payloadFieldsRepository.GetStatus(statusName)
	// __AUTO_GENERATED_PRINT_VAR_START__
	fmt.Println(fmt.Sprintf("HIT testStatusCaching payloadReturned: %v", payloadReturned)) // __AUTO_GENERATED_PRINT_VAR_END__

	Expect(mockPayloadFieldsRepository.getStatusCalled).To(Equal(false))
	Expect(payloadReturned.Id).To(Equal(int32(1234)))
	Expect(payloadReturned.Name).To(Equal(statusName))
}

func testServiceCaching(payloadFieldsRepository PayloadFieldsRepository, mockPayloadFieldsRepository *mockPayloadFieldsRepository, serviceName string) {
	// Cache miss
	payloadReturned := payloadFieldsRepository.GetService(serviceName)

	Expect(mockPayloadFieldsRepository.getServiceCalled).To(Equal(true))
	Expect(payloadReturned.Id).To(Equal(int32(1234)))
	Expect(payloadReturned.Name).To(Equal(serviceName))

	// Cache hit
	mockPayloadFieldsRepository.getServiceCalled = false
	payloadReturned = payloadFieldsRepository.GetService(serviceName)

	Expect(mockPayloadFieldsRepository.getStatusCalled).To(Equal(false))
	Expect(payloadReturned.Id).To(Equal(int32(1234)))
	Expect(payloadReturned.Name).To(Equal(serviceName))
}

func testSourceCaching(payloadFieldsRepository PayloadFieldsRepository, mockPayloadFieldsRepository *mockPayloadFieldsRepository, sourceName string) {
	// Cache miss
	payloadReturned := payloadFieldsRepository.GetSource(sourceName)

	Expect(mockPayloadFieldsRepository.getSourceCalled).To(Equal(true))
	Expect(payloadReturned.Id).To(Equal(int32(1234)))
	Expect(payloadReturned.Name).To(Equal(sourceName))

	// Cache hit
	mockPayloadFieldsRepository.getSourceCalled = false
	payloadReturned = payloadFieldsRepository.GetSource(sourceName)

	Expect(mockPayloadFieldsRepository.getSourceCalled).To(Equal(false))
	Expect(payloadReturned.Id).To(Equal(int32(1234)))
	Expect(payloadReturned.Name).To(Equal(sourceName))
}

var _ = Describe("Queries", func() {
	db := test.WithDatabase()
	redis.Init()

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
		mockPayloadFieldsRepository := &mockPayloadFieldsRepository{}

		payloadFieldsRepository, err := newPayloadFieldsRepositoryFromCache(mockPayloadFieldsRepository)
		if err != nil {
			panic(err)
		}

		testStatusCaching(payloadFieldsRepository, mockPayloadFieldsRepository, statusName)
	})
	It("Checks if we got a cached service result from the database", func() {
		const serviceName = "TestService"
		mockPayloadFieldsRepository := &mockPayloadFieldsRepository{}

		payloadFieldsRepository, err := newPayloadFieldsRepositoryFromCache(mockPayloadFieldsRepository)
		if err != nil {
			panic(err)
		}

		testServiceCaching(payloadFieldsRepository, mockPayloadFieldsRepository, serviceName)
	})
	It("Checks if we got a cached source result from the database", func() {
		const sourceName = "TestSource"
		mockPayloadFieldsRepository := &mockPayloadFieldsRepository{}

		payloadFieldsRepository, err := newPayloadFieldsRepositoryFromCache(mockPayloadFieldsRepository)
		if err != nil {
			panic(err)
		}

		testSourceCaching(payloadFieldsRepository, mockPayloadFieldsRepository, sourceName)
	})
	It("Checks if we got a status result from redis", func() {
		const statusName string = "TestStatusRedis"
		fmt.Println("STARTING REDIS STATUS TEST")
		mockPayloadFieldsRepository := &mockPayloadFieldsRepository{}

		payloadFieldsRepository, err := newPayloadFieldsRepositoryFromRedis(mockPayloadFieldsRepository)
		if err != nil {
			panic(err)
		}
		testStatusCaching(payloadFieldsRepository, mockPayloadFieldsRepository, statusName)
		fmt.Println("ENDING REDIS STATUS TEST")
	})
	It("Checks if we got a service result from redis", func() {
		const serviceName string = "TestServiceRedis"
		mockPayloadFieldsRepository := &mockPayloadFieldsRepository{}

		payloadFieldsRepository, err := newPayloadFieldsRepositoryFromRedis(mockPayloadFieldsRepository)
		if err != nil {
			panic(err)
		}
		testServiceCaching(payloadFieldsRepository, mockPayloadFieldsRepository, serviceName)
	})
	It("Checks if we got a source result from redis", func() {
		const sourceName string = "TestSourceRedis"
		mockPayloadFieldsRepository := &mockPayloadFieldsRepository{}

		payloadFieldsRepository, err := newPayloadFieldsRepositoryFromRedis(mockPayloadFieldsRepository)
		if err != nil {
			panic(err)
		}
		testSourceCaching(payloadFieldsRepository, mockPayloadFieldsRepository, sourceName)
	})
})
