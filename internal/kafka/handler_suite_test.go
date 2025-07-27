package kafka

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
)

func TestEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	l.InitLogger()
	RunSpecs(t, "Kafka Message Handler Suite")
}
