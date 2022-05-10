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
			RequestId: requestId,
			Account: "1234",
			InventoryId: getUUID(),
			SystemId: getUUID(),
		}
		Expect(db().Create(&payload).Error).ToNot(HaveOccurred())

		payload, err := GetPayloadByRequestId(db(), requestId)

		Expect(err).ToNot(HaveOccurred())
		
		Expect(payload.RequestId).To(Equal(requestId))
		Expect(payload.Account).To(Equal("1234"))
	})
})
