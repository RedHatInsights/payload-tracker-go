package kafka

import (
	"context"
	"encoding/json"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/models/message"
	"github.com/redhatinsights/payload-tracker-go/internal/queries"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

func newKafkaMessage(value message.PayloadStatusMessage) *k.Message {
	msgValue, err := json.Marshal(value)
	Expect(err).ToNot(HaveOccurred())

	topic := "topic.payload.status"

	return &k.Message{
		Value: msgValue,
		TopicPartition: k.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    k.Offset(0),
		},
	}
}

func getSimplePayloadStatusMessage() message.PayloadStatusMessage {
	date, _ := time.Parse(time.RFC3339, "2022-06-07T11:00:10.356Z")
	return message.PayloadStatusMessage{
		Service:     "puptoo",
		Source:      "test-source",
		Account:     "1234",
		OrgID:       "5678",
		RequestID:   "e4b3d38f199f4abdb1cfbcf6e3b81f56",
		InventoryID: "0e71e590-19e8-456e-8439-6cec9a1ae074",
		SystemID:    "ef49a293-64f3-4945-9797-fc9fe6ec73e1",
		Status:      "success",
		StatusMSG:   "done",
		Date:        message.FormatedTime{date},
	}
}

var _ = Describe("Kafka message handler", func() {
	var msgHandler handler

	db := test.WithDatabase()

	BeforeEach(func() {
		msgHandler = handler{
			db: db(),
		}
	})

	Describe("On valid payload status message", func() {
		It("Creates the required DB entries", func() {
			payloadMsgVal := getSimplePayloadStatusMessage()
			payloadStatusMessage := newKafkaMessage(payloadMsgVal)

			msgHandler.onMessage(context.Background(), payloadStatusMessage, config.Get())

			dbResult := queries.RetrieveRequestIdPayloads(db(), payloadMsgVal.RequestID, "created_at", "asc", "0")

			Expect(dbResult[0].Service).To(Equal(payloadMsgVal.Service))
			Expect(dbResult[0].Account).To(Equal(payloadMsgVal.Account))
			Expect(dbResult[0].OrgID).To(Equal(payloadMsgVal.OrgID))
			Expect(dbResult[0].RequestID).To(Equal(payloadMsgVal.RequestID))
			Expect(dbResult[0].InventoryID).To(Equal(payloadMsgVal.InventoryID))
			Expect(dbResult[0].SystemID).To(Equal(payloadMsgVal.SystemID))
			Expect(dbResult[0].Status).To(Equal(payloadMsgVal.Status))
			Expect(dbResult[0].StatusMsg).To(Equal(payloadMsgVal.StatusMSG))
			Expect(dbResult[0].Source).To(Equal(payloadMsgVal.Source))
		})
	})

	Describe("On valid request ID", func() {
		It("Succeeds and returns true", func() {
			requestID := "e4b3d38f199f4abdb1cfbcf6e3b81f56"

			validationResult := validateRequestID(32, requestID)

			Expect(validationResult).To(Equal(true))
		})
	})

	Describe("On invalid request ID", func() {
		It("Fails on a request ID of invalid length", func() {
			requestID := uuid.New().String() // Default max request id length in 32 (equal to UUID without any dashes). This produces an UUID with dashes. e.g. > 32

			validationResult := validateRequestID(32, requestID)

			Expect(validationResult).To(Equal(false))
		})

		It("Fails on undeclared request ID", func() {
			requestID := ""

			validationResult := validateRequestID(32, requestID)

			Expect(validationResult).To(Equal(false))
		})

		It("Does not create db entries", func() {
			payloadMsgVal := getSimplePayloadStatusMessage()
			payloadMsgVal.RequestID = uuid.New().String() // Default max request id length in 32 (equal to UUID without any dashes). This produces an UUID with dashes. e.g. > 32

			payloadStatusMessage := newKafkaMessage(payloadMsgVal)

			msgHandler.onMessage(context.Background(), payloadStatusMessage, config.Get())

			dbResult := queries.RetrieveRequestIdPayloads(db(), payloadMsgVal.RequestID, "created_at", "asc", "0")

			Expect(len(dbResult)).To(Equal(0))
		})
	})
})
